package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var flagAddr = flag.String("addr", "127.0.0.1:7777", "address to listen on (host:port)")

var (
	wsUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// HTML widget markers for PTY output parsing
	htmlStartMarker = []byte("\x1b]9001;HTML_START\x07")
	htmlEndMarker   = []byte("\x1b]9001;HTML_END\x07")
)

// Default PTY size
const (
	defaultPTYRows = 24
	defaultPTYCols = 80
)

// Widget represents a tracked widget session.
type Widget struct {
	ID    string
	State json.RawMessage
}

// Refresh triggers widget-specific refresh logic.
func (w *Widget) Refresh() {
	RefreshWidget(w.ID)
}

// WidgetActionRequest models /widget/{id}/action payloads.
type WidgetActionRequest struct {
	Action string          `json:"action"`
	Type   string          `json:"type"`
	Cmd    string          `json:"cmd"`
	State  json.RawMessage `json:"state"`
}

// ShellServer manages the single PTY-backed shell and HTTP handlers.
type ShellServer struct {
	ptyFile *os.File
	ptyMu   sync.Mutex

	clients      map[*websocket.Conn]struct{}
	clientsMu    sync.RWMutex
	connWriteMu  map[*websocket.Conn]*sync.Mutex // Per-connection write mutex
	connWriteMuM sync.Mutex                       // Mutex for connWriteMu map

	widgets   map[string]*Widget
	widgetsMu sync.RWMutex

	htmlWidgets   map[int]string // Stores HTML content by widget ID
	htmlWidgetsMu sync.RWMutex
	htmlCounter   int

	buffer   []byte
	bufferMu sync.Mutex

	htmlBuffer []byte // Accumulates incomplete HTML blocks across PTY reads
	htmlBufMu  sync.Mutex

	shellPGID int // The shell's process group ID (idle state)
}

// getForegroundPGID gets the current foreground process group ID
func getForegroundPGID(fd uintptr) (int, error) {
	var pgid int
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&pgid)))
	if errno != 0 {
		return 0, errno
	}
	return pgid, nil
}

// startPTY creates a new PTY running zsh with the standard environment.
// Returns the pty file and the shell's process group ID.
func startPTY() (*os.File, int, error) {
	cmd := exec.Command("zsh", "-l")
	goshellHome, _ := os.Getwd()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "GOSHELL_HOME="+goshellHome)

	ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: defaultPTYRows,
		Cols: defaultPTYCols,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("start zsh pty: %w", err)
	}

	// Wait a bit for shell to start, then capture its PGID
	time.Sleep(100 * time.Millisecond)
	shellPGID, err := getForegroundPGID(ptyFile.Fd())
	if err != nil {
		ptyFile.Close()
		return nil, 0, fmt.Errorf("get shell PGID: %w", err)
	}

	return ptyFile, shellPGID, nil
}

func newShellServer() (*ShellServer, error) {
	ptyFile, shellPGID, err := startPTY()
	if err != nil {
		return nil, err
	}

	server := &ShellServer{
		ptyFile:     ptyFile,
		clients:     make(map[*websocket.Conn]struct{}),
		connWriteMu: make(map[*websocket.Conn]*sync.Mutex),
		widgets:     make(map[string]*Widget),
		htmlWidgets: make(map[int]string),
		shellPGID:   shellPGID,
	}

	go server.streamPTY()
	go server.monitorStatus()
	return server, nil
}

// containsAltScreenExit checks if data contains escape sequences that exit alternate screen buffer
func containsAltScreenExit(data []byte) bool {
	// Common sequences for exiting alternate screen:
	// ESC [ ? 1049 l  (xterm)
	// ESC [ ? 47 l    (older xterm)
	// ESC [ ? 1047 l  (another variant)
	patterns := [][]byte{
		[]byte("\x1b[?1049l"),
		[]byte("\x1b[?47l"),
		[]byte("\x1b[?1047l"),
	}
	for _, pattern := range patterns {
		if bytes.Contains(data, pattern) {
			return true
		}
	}
	return false
}

// extractAndStoreHTML extracts HTML content from accumulated PTY data and stores it
// Returns: (processedData, remainingBuffer, widgetIDs)
// - processedData: data with HTML blocks replaced by links
// - remainingBuffer: incomplete HTML block data to keep for next read
// - widgetIDs: IDs of extracted widgets
func (s *ShellServer) extractAndStoreHTML(data []byte) ([]byte, []byte, []int) {
	result := data
	var widgetIDs []int

	for {
		startIdx := bytes.Index(result, htmlStartMarker)
		if startIdx == -1 {
			// No HTML_START found, return all data as processed
			return result, nil, widgetIDs
		}

		endIdx := bytes.Index(result[startIdx:], htmlEndMarker)
		if endIdx == -1 {
			// Found HTML_START but no HTML_END - keep this for next read
			return result[:startIdx], result[startIdx:], widgetIDs
		}

		// Extract the HTML content
		htmlContentStart := startIdx + len(htmlStartMarker)
		htmlContentEnd := startIdx + endIdx
		htmlContent := result[htmlContentStart:htmlContentEnd]

		// Store the HTML content with a unique ID
		s.htmlWidgetsMu.Lock()
		s.htmlCounter++
		widgetID := s.htmlCounter
		s.htmlWidgets[widgetID] = string(htmlContent)
		s.htmlWidgetsMu.Unlock()

		widgetIDs = append(widgetIDs, widgetID)

		// Create a clickable link using OSC 8 hyperlinks
		linkText := fmt.Sprintf("View HTML Output #%d", widgetID)
		replacement := []byte(fmt.Sprintf("\x1b]8;;htmlwidget:%d\x07\x1b[34;4m%s\x1b[0m\x1b]8;;\x07",
			widgetID, linkText))

		// Replace from HTML_START to HTML_END with the link
		endIdx += startIdx + len(htmlEndMarker)
		result = append(result[:startIdx], append(replacement, result[endIdx:]...)...)
	}
}

// stripHTMLMode removes HTML mode sequences from buffer (used for cleaning up buffer)
func stripHTMLMode(data []byte) []byte {
	result := data

	for {
		startIdx := bytes.Index(result, htmlStartMarker)
		if startIdx == -1 {
			break
		}

		endIdx := bytes.Index(result[startIdx:], htmlEndMarker)
		if endIdx == -1 {
			// No matching end, strip from start to end of buffer
			result = result[:startIdx]
			break
		}

		// Just remove the HTML block entirely
		endIdx += startIdx + len(htmlEndMarker)
		result = append(result[:startIdx], result[endIdx:]...)
	}

	return result
}

func (s *ShellServer) restart() error {
	s.ptyMu.Lock()
	if s.ptyFile != nil {
		s.ptyFile.Close()
	}
	s.ptyMu.Unlock()

	ptyFile, shellPGID, err := startPTY()
	if err != nil {
		return err
	}

	s.ptyMu.Lock()
	s.ptyFile = ptyFile
	s.shellPGID = shellPGID
	s.ptyMu.Unlock()

	s.bufferMu.Lock()
	s.buffer = nil
	s.bufferMu.Unlock()

	go s.streamPTY()
	go s.monitorStatus()
	return nil
}

func (s *ShellServer) streamPTY() {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptyFile.Read(buf)
		if n > 0 {
			data := buf[:n]

			// Append to HTML buffer to handle HTML content split across reads
			s.htmlBufMu.Lock()
			s.htmlBuffer = append(s.htmlBuffer, data...)

			// Try to extract complete HTML blocks from the accumulated buffer
			processedData, remainingBuf, widgetIDs := s.extractAndStoreHTML(s.htmlBuffer)

			// Keep any incomplete HTML block for next read
			s.htmlBuffer = remainingBuf
			s.htmlBufMu.Unlock()

			if len(widgetIDs) > 0 {
				log.Printf("DEBUG: Extracted %d HTML widgets, processed data length: %d bytes", len(widgetIDs), len(processedData))
				previewLen := 200
				if len(processedData) < previewLen {
					previewLen = len(processedData)
				}
				log.Printf("DEBUG: First %d bytes of processed data: %q", previewLen, string(processedData[:previewLen]))
			}

			s.bufferMu.Lock()
			// If we're exiting alternate screen buffer, clear the history
			// since that content is no longer visible
			if containsAltScreenExit(data) {
				s.buffer = nil
			}
			// Add processed data (with links instead of HTML) to buffer
			s.buffer = append(s.buffer, processedData...)
			// Clean up any HTML sequences that might be in the buffer
			s.buffer = stripHTMLMode(s.buffer)
			if len(s.buffer) > 64*1024 {
				s.buffer = s.buffer[len(s.buffer)-64*1024:]
			}
			s.bufferMu.Unlock()

			// Broadcast processed data (with links) to all clients
			s.broadcast(processedData)

			// Notify live clients about new HTML widgets so they auto-display
			for _, widgetID := range widgetIDs {
				s.broadcastHTMLNotification(widgetID)
			}
		}
		if err != nil {
			log.Printf("pty read error: %v", err)
			return
		}
	}
}

// broadcastMessage sends a message to all connected clients.
// If unregisterOnError is true, failed connections are unregistered.
func (s *ShellServer) broadcastMessage(msgType int, data []byte, unregisterOnError bool) {
	s.clientsMu.RLock()
	conns := make([]*websocket.Conn, 0, len(s.clients))
	for conn := range s.clients {
		conns = append(conns, conn)
	}
	s.clientsMu.RUnlock()

	for _, conn := range conns {
		s.connWriteMuM.Lock()
		mu, ok := s.connWriteMu[conn]
		s.connWriteMuM.Unlock()

		if !ok {
			continue
		}

		mu.Lock()
		err := conn.WriteMessage(msgType, data)
		mu.Unlock()

		if err != nil {
			log.Printf("websocket write error: %v", err)
			if unregisterOnError {
				s.unregisterClient(conn)
			}
		}
	}
}

func (s *ShellServer) broadcast(data []byte) {
	s.broadcastMessage(websocket.BinaryMessage, data, true)
}

func (s *ShellServer) broadcastStatus(state string) {
	msg := map[string]string{"kind": "status", "state": state}
	data, _ := json.Marshal(msg)
	s.broadcastMessage(websocket.TextMessage, data, false)
}

func (s *ShellServer) broadcastHTMLNotification(widgetID int) {
	msg := map[string]any{"kind": "html", "widget_id": widgetID}
	data, _ := json.Marshal(msg)
	s.broadcastMessage(websocket.TextMessage, data, false)
}

func (s *ShellServer) monitorStatus() {
	lastState := "waiting"
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		s.ptyMu.Lock()
		if s.ptyFile == nil {
			s.ptyMu.Unlock()
			return
		}
		pgid, err := getForegroundPGID(s.ptyFile.Fd())
		s.ptyMu.Unlock()

		if err != nil {
			continue
		}

		var newState string
		if pgid == s.shellPGID {
			newState = "waiting"
		} else {
			newState = "running"
		}

		if newState != lastState {
			s.broadcastStatus(newState)
			lastState = newState
		}
	}
}

func (s *ShellServer) writeToPTY(data []byte) error {
	s.ptyMu.Lock()
	defer s.ptyMu.Unlock()
	_, err := s.ptyFile.Write(data)
	return err
}

func (s *ShellServer) addClient(conn *websocket.Conn) {
	// Create a write mutex for this connection
	s.connWriteMuM.Lock()
	s.connWriteMu[conn] = &sync.Mutex{}
	mu := s.connWriteMu[conn]
	s.connWriteMuM.Unlock()

	s.clientsMu.Lock()
	s.clients[conn] = struct{}{}
	s.clientsMu.Unlock()

	s.bufferMu.Lock()
	buffered := make([]byte, len(s.buffer))
	copy(buffered, s.buffer)
	s.bufferMu.Unlock()

	mu.Lock()
	if len(buffered) > 0 {
		conn.WriteMessage(websocket.BinaryMessage, buffered)
	}
	// Signal that server is ready and all buffered content has been sent
	conn.WriteMessage(websocket.TextMessage, []byte(`{"kind":"ready"}`))
	mu.Unlock()
}

func (s *ShellServer) unregisterClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	delete(s.clients, conn)
	s.clientsMu.Unlock()

	s.connWriteMuM.Lock()
	delete(s.connWriteMu, conn)
	s.connWriteMuM.Unlock()

	conn.Close()
}

func (s *ShellServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}

func (s *ShellServer) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := s.restart(); err != nil {
		log.Printf("restart error: %v", err)
		http.Error(w, "failed to restart shell", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *ShellServer) handleResize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var size struct {
		Rows uint16 `json:"rows"`
		Cols uint16 `json:"cols"`
	}

	if err := json.NewDecoder(r.Body).Decode(&size); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	s.ptyMu.Lock()
	defer s.ptyMu.Unlock()

	if err := pty.Setsize(s.ptyFile, &pty.Winsize{
		Rows: size.Rows,
		Cols: size.Cols,
	}); err != nil {
		log.Printf("resize error: %v", err)
		http.Error(w, "failed to resize terminal", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *ShellServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	s.addClient(conn)
	defer s.unregisterClient(conn)

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("websocket read error: %v", err)
			return
		}
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			continue
		}
		if err := s.writeToPTY(data); err != nil {
			log.Printf("pty write error: %v", err)
			return
		}
	}
}

func (s *ShellServer) handleWidgetAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := widgetIDFromPath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	defer r.Body.Close()
	var payload WidgetActionRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	switch payload.Type {
	case "shell":
		if payload.Cmd == "" {
			http.Error(w, "cmd required for shell action", http.StatusBadRequest)
			return
		}
		cmd := append([]byte(payload.Cmd), '\n')
		if err := s.writeToPTY(cmd); err != nil {
			http.Error(w, "failed to write to shell", http.StatusInternalServerError)
			return
		}
	case "internal":
		widget := s.updateWidgetState(id, payload.State)
		widget.Refresh()
	default:
		http.Error(w, "unsupported widget type", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *ShellServer) handleHTMLWidget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Extract widget ID from path: /htmlwidget/123
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/htmlwidget/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	var widgetID int
	if _, err := fmt.Sscanf(parts[0], "%d", &widgetID); err != nil {
		http.NotFound(w, r)
		return
	}

	s.htmlWidgetsMu.RLock()
	htmlContent, ok := s.htmlWidgets[widgetID]
	s.htmlWidgetsMu.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent))
}

func widgetIDFromPath(path string) (string, error) {
	const prefix = "/widget/"
	if !strings.HasPrefix(path, prefix) {
		return "", errors.New("invalid path")
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "action" || parts[0] == "" {
		return "", errors.New("invalid widget path")
	}
	return parts[0], nil
}

func (s *ShellServer) updateWidgetState(id string, state json.RawMessage) *Widget {
	s.widgetsMu.Lock()
	defer s.widgetsMu.Unlock()
	widget, ok := s.widgets[id]
	if !ok {
		widget = &Widget{ID: id}
		s.widgets[id] = widget
	}
	if len(state) > 0 {
		copied := make(json.RawMessage, len(state))
		copy(copied, state)
		widget.State = copied
	}
	return widget
}

// RefreshWidget describes where real DOM patch logic will live later.
func RefreshWidget(id string) {
	log.Printf("widget %s refreshed", id)
}

func main() {
	flag.Parse()

	server, err := newShellServer()
	if err != nil {
		log.Fatalf("create shell server: %v", err)
	}

	http.HandleFunc("/", server.handleIndex)
	http.Handle("/js/", http.StripPrefix("/", http.FileServer(http.Dir("web"))))
	http.Handle("/css/", http.StripPrefix("/", http.FileServer(http.Dir("web"))))
	http.HandleFunc("/ws/shell", server.handleWebSocket)
	http.HandleFunc("/restart", server.handleRestart)
	http.HandleFunc("/resize", server.handleResize)
	http.HandleFunc("/widget/", server.handleWidgetAction)
	http.HandleFunc("/htmlwidget/", server.handleHTMLWidget)

	log.Printf("server listening on http://%s", *flagAddr)
	if err := http.ListenAndServe(*flagAddr, nil); err != nil {
		log.Fatalf("http server stopped: %v", err)
	}
}
