package main

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/gorilla/websocket"
)

func startTestServer(t *testing.T) (*ShellServer, *httptest.Server) {
    t.Helper()
    s, err := newShellServer()
    if err != nil {
        t.Fatalf("newShellServer: %v", err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/ws/shell", s.handleWebSocket)
    mux.HandleFunc("/widget/", s.handleWidgetAction)

    ts := httptest.NewServer(mux)
    return s, ts
}

func wsURLFromHTTP(httpURL string, path string) string {
    // convert http://127.0.0.1:12345 -> ws://127.0.0.1:12345
    if strings.HasPrefix(httpURL, "https://") {
        return "wss://" + strings.TrimPrefix(httpURL, "https://") + path
    }
    return "ws://" + strings.TrimPrefix(httpURL, "http://") + path
}

func TestPTYToClient(t *testing.T) {
    s, ts := startTestServer(t)
    defer ts.Close()
    defer s.ptyFile.Close()

    url := wsURLFromHTTP(ts.URL, "/ws/shell")
    dialer := websocket.Dialer{}
    conn, _, err := dialer.Dial(url, nil)
    if err != nil {
        t.Fatalf("dial ws: %v", err)
    }
    defer conn.Close()

    // write a command directly to the PTY
    if err := s.writeToPTY([]byte("echo test-hello-from-server\n")); err != nil {
        t.Fatalf("writeToPTY: %v", err)
    }

    // expect to receive the echoed text via websocket broadcast
    conn.SetReadDeadline(time.Now().Add(3 * time.Second))
    _, msg, err := conn.ReadMessage()
    if err != nil {
        t.Fatalf("read ws: %v", err)
    }
    if !strings.Contains(string(msg), "test-hello-from-server") {
        t.Fatalf("expected message to contain test-hello-from-server; got: %q", string(msg))
    }
}

func TestClientToPTY(t *testing.T) {
    s, ts := startTestServer(t)
    defer ts.Close()
    defer s.ptyFile.Close()

    url := wsURLFromHTTP(ts.URL, "/ws/shell")
    dialer := websocket.Dialer{}
    conn, _, err := dialer.Dial(url, nil)
    if err != nil {
        t.Fatalf("dial ws: %v", err)
    }
    defer conn.Close()

    // send a command from the client which the server should write into PTY
    if err := conn.WriteMessage(websocket.TextMessage, []byte("echo message-from-client\n")); err != nil {
        t.Fatalf("write msg: %v", err)
    }

    // read broadcasted output and assert it contains our echoed string
    conn.SetReadDeadline(time.Now().Add(3 * time.Second))
    _, msg, err := conn.ReadMessage()
    if err != nil {
        t.Fatalf("read ws: %v", err)
    }
    if !strings.Contains(string(msg), "message-from-client") {
        t.Fatalf("expected message to contain message-from-client; got: %q", string(msg))
    }
}
