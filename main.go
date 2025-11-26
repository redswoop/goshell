package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

const listenAddr = "127.0.0.1:7777"

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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

	clients   map[*websocket.Conn]struct{}
	clientsMu sync.RWMutex

	widgets   map[string]*Widget
	widgetsMu sync.RWMutex

	buffer   []byte
	bufferMu sync.Mutex

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

func newShellServer() (*ShellServer, error) {
	cmd := exec.Command("zsh", "-l")
	// Add lsh directory to PATH
	lshPath, _ := os.Getwd()
	newPath := fmt.Sprintf("%s/lsh:%s", lshPath, os.Getenv("PATH"))
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "PATH="+newPath)

	ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})
	if err != nil {
		return nil, fmt.Errorf("start zsh pty: %w", err)
	}

	// Wait a bit for shell to start, then capture its PGID
	time.Sleep(100 * time.Millisecond)
	shellPGID, err := getForegroundPGID(ptyFile.Fd())
	if err != nil {
		ptyFile.Close()
		return nil, fmt.Errorf("get shell PGID: %w", err)
	}

	server := &ShellServer{
		ptyFile:   ptyFile,
		clients:   make(map[*websocket.Conn]struct{}),
		widgets:   make(map[string]*Widget),
		shellPGID: shellPGID,
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

func (s *ShellServer) restart() error {
	s.ptyMu.Lock()
	if s.ptyFile != nil {
		s.ptyFile.Close()
	}

	cmd := exec.Command("zsh", "-l")
	// Add lsh directory to PATH
	lshPath, _ := os.Getwd()
	newPath := fmt.Sprintf("%s/lsh:%s", lshPath, os.Getenv("PATH"))
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "PATH="+newPath)

	ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})
	if err != nil {
		s.ptyMu.Unlock()
		return fmt.Errorf("restart zsh pty: %w", err)
	}

	s.ptyFile = ptyFile
	s.ptyMu.Unlock()

	// Wait for shell to start, then capture new PGID
	time.Sleep(100 * time.Millisecond)
	shellPGID, err := getForegroundPGID(ptyFile.Fd())
	if err != nil {
		return fmt.Errorf("get shell PGID after restart: %w", err)
	}
	s.shellPGID = shellPGID

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
			s.bufferMu.Lock()
			// If we're exiting alternate screen buffer, clear the history
			// since that content is no longer visible
			if containsAltScreenExit(data) {
				s.buffer = nil
			}
			s.buffer = append(s.buffer, data...)
			if len(s.buffer) > 64*1024 {
				s.buffer = s.buffer[len(s.buffer)-64*1024:]
			}
			s.bufferMu.Unlock()
			s.broadcast(data)
		}
		if err != nil {
			log.Printf("pty read error: %v", err)
			return
		}
	}
}

func (s *ShellServer) broadcast(data []byte) {
	s.clientsMu.RLock()
	conns := make([]*websocket.Conn, 0, len(s.clients))
	for conn := range s.clients {
		conns = append(conns, conn)
	}
	s.clientsMu.RUnlock()

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("websocket write error: %v", err)
			s.unregisterClient(conn)
		}
	}
}

func (s *ShellServer) broadcastStatus(state string) {
	s.clientsMu.RLock()
	conns := make([]*websocket.Conn, 0, len(s.clients))
	for conn := range s.clients {
		conns = append(conns, conn)
	}
	s.clientsMu.RUnlock()

	msg := map[string]string{"kind": "status", "state": state}
	data, _ := json.Marshal(msg)

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("websocket status write error: %v", err)
		}
	}
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
	s.clientsMu.Lock()
	s.clients[conn] = struct{}{}
	s.clientsMu.Unlock()

	s.bufferMu.Lock()
	buffered := make([]byte, len(s.buffer))
	copy(buffered, s.buffer)
	s.bufferMu.Unlock()

	if len(buffered) > 0 {
		conn.WriteMessage(websocket.BinaryMessage, buffered)
	}
}

func (s *ShellServer) unregisterClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	delete(s.clients, conn)
	s.clientsMu.Unlock()
	conn.Close()
}

func (s *ShellServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "index.html")
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
	server, err := newShellServer()
	if err != nil {
		log.Fatalf("create shell server: %v", err)
	}

	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/ws/shell", server.handleWebSocket)
	http.HandleFunc("/restart", server.handleRestart)
	http.HandleFunc("/resize", server.handleResize)
	http.HandleFunc("/widget/", server.handleWidgetAction)

	log.Printf("server listening on http://%s", listenAddr)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("http server stopped: %v", err)
	}
}
