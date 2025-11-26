# goshell

A lightweight web-based terminal built with Go and xterm.js that persists your shell session across browser reconnections. Supports custom escape sequences for rendering rich HTML content directly in the terminal.

## What It Does

goshell runs a single persistent shell (zsh) session on the server and allows multiple browser clients to connect and interact with it simultaneously. If you refresh your browser or reconnect, you're dropped right back into the same shell session with its history intact.

The terminal supports a custom HTML rendering mode via escape sequences, allowing programs to display rich interactive content (like file browsers with clickable sort buttons) instead of plain text.

## Key Features

- **Persistent shell sessions**: Your shell keeps running even when you close the browser
- **Session history replay**: Reconnecting clients receive the last 64KB of terminal output
- **Smart buffer management**: Automatically clears the replay buffer when full-screen apps (like vim) exit to prevent escape sequence junk
- **Multi-client support**: Multiple browsers can connect to the same shell simultaneously
- **Live status indicator**: Shows whether the shell is waiting for input or running a command
- **Terminal resizing**: Automatically syncs terminal dimensions with the PTY
- **HTML rendering mode**: Custom escape sequences allow programs to render interactive HTML content
- **Widget system**: HTML content can trigger shell commands via a widget action API

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

### HTML Rendering Mode

The terminal supports a dual-protocol system for rendering both VT100 terminal output and rich HTML content:

**Custom Escape Sequences:**

- `ESC]9001;HTML_START\x07` - Begins HTML mode
- `ESC]9001;HTML_END\x07` - Ends HTML mode and renders the accumulated HTML

Programs can use these OSC (Operating System Command) sequences to inject HTML into a dedicated panel above the terminal. The HTML panel:

- Appears above the xterm.js terminal with smooth transitions
- Can be toggled with a "Show/Hide HTML" button in the header
- Automatically resizes the terminal to maintain proper dimensions
- Supports full CSS styling and JavaScript interactions

**Widget Action API:**
HTML content can execute shell commands via the `window.runCommand(cmd)` JavaScript function, which sends commands to `/widget/{id}/action` endpoint. Commands are executed in the persistent shell session.

### Client Side

The browser client (`index.html`) uses xterm.js to provide a full-featured terminal emulator:

- Connects via WebSocket to `/ws/shell`
- Receives binary messages containing PTY output
- Receives text messages containing status updates
- Parses custom escape sequences to detect HTML mode
- Sends user keystrokes back to the server
- Handles terminal resizing and restoration
- Displays shell status (waiting/running)
- Renders HTML content in a dedicated panel

## Example: lsh (HTML-aware ls)

The `lsh` directory contains an example program that demonstrates the HTML rendering capabilities. It's an enhanced `ls` command that displays directory contents as interactive cards with sorting buttons.

**Features:**

- Displays files and directories as styled cards with icons
- Shows file size and permissions
- Interactive sort buttons (Name, Date, Size, Reverse)
- Sort buttons execute commands in the shell to refresh the view
- Uses absolute paths so it works regardless of current directory

**Running lsh:**

```bash
lsh [directory]
lsh -t [directory]  # Sort by modification time
lsh -S [directory]  # Sort by size
lsh -r [directory]  # Reverse sort order
```

The `lsh` binary is automatically added to the shell's PATH when the server starts.

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
