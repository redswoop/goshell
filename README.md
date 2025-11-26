# goshell

A lightweight web-based terminal built with Go and xterm.js that persists your shell session across browser reconnections.

## What It Does

goshell runs a single persistent shell (zsh) session on the server and allows multiple browser clients to connect and interact with it simultaneously. If you refresh your browser or reconnect, you're dropped right back into the same shell session with its history intact.

## Key Features

- **Persistent shell sessions**: Your shell keeps running even when you close the browser
- **Session history replay**: Reconnecting clients receive the last 64KB of terminal output
- **Smart buffer management**: Automatically clears the replay buffer when full-screen apps (like vim) exit to prevent escape sequence junk
- **Multi-client support**: Multiple browsers can connect to the same shell simultaneously
- **Live status indicator**: Shows whether the shell is waiting for input or running a command
- **Terminal resizing**: Automatically syncs terminal dimensions with the PTY

## How It Works

### Server Architecture

The Go server (`main.go`) manages a single PTY-backed zsh process:

1. **PTY Management**: Creates a pseudo-terminal using `github.com/creack/pty` and spawns a zsh shell
2. **Output Buffering**: Maintains a rolling 64KB buffer of terminal output for replay to new connections
3. **WebSocket Broadcasting**: All PTY output is broadcast to connected WebSocket clients in real-time
4. **Process Monitoring**: Tracks the foreground process group ID to detect when commands are running vs. idle

### Smart Buffer Clearing

When you run full-screen terminal applications like vim, they use VT escape sequences to switch to an alternate screen buffer. On exit, these apps send sequences like `ESC[?1049l` to restore the normal screen.

The server detects these alternate screen exit sequences and clears the replay buffer, since:
- The alternate screen content is no longer visible
- Replaying these escape sequences out of context causes garbled output

### Client Side

The browser client (`index.html`) uses xterm.js to provide a full-featured terminal emulator:

- Connects via WebSocket to `/ws/shell`
- Receives binary messages containing PTY output
- Sends user keystrokes back to the server
- Handles terminal resizing and restoration
- Displays shell status (waiting/running)

## Running

```bash
go run main.go
```

Then open your browser to `http://127.0.0.1:7777`

## Dependencies

- `github.com/creack/pty` - PTY management
- `github.com/gorilla/websocket` - WebSocket server
- xterm.js (loaded via CDN) - Terminal emulator

## API Endpoints

- `GET /` - Serves the HTML terminal interface
- `GET /ws/shell` - WebSocket endpoint for terminal I/O
- `POST /restart` - Restart the shell session (clears buffer)
- `POST /resize` - Resize the PTY (receives `{rows, cols}`)
- `POST /widget/{id}/action` - Widget action handler (future extensibility)
