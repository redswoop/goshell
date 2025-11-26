# Testing Guide

## Running the Test Suite

1. Start the server:
   ```bash
   go run main.go
   ```

2. Open browser to http://127.0.0.1:7777

3. Run the interactive test:
   ```bash
   ./test_shell.sh
   ```

## Expected Behaviors

### Test 1: Regular Output
- Type: `echo 'Hello World'`
- Expected: Text appears in terminal normally
- On refresh: Text is replayed from buffer

### Test 2 & 3: HTML Command (Live)
- Type: `lsh`
- Expected in terminal: Blue clickable link "View HTML Output #1"
- Expected in HTML pane: File listing with sort buttons automatically appears
- Behavior: Link appears AND HTML auto-displays

### Test 4: After Refresh
- Action: Refresh browser (Cmd+R)
- Expected: Link "View HTML Output #1" visible in terminal history
- Expected: HTML pane is hidden/empty (no auto-display on replay)

### Test 5: Click Link After Refresh
- Action: Click "View HTML Output #1"
- Expected: HTML pane appears with file listing
- Behavior: Same HTML as before

### Test 6: Interactive Buttons
- Action: Click "Date" sort button in HTML pane
- Expected: New link appears "View HTML Output #2"
- Expected: HTML pane updates with re-sorted listing
- Behavior: Each click creates new HTML widget

### Test 7: Full-Screen Apps
- Type: `vim` then `:q`
- Expected: vim works normally
- Expected: After exit, buffer is cleared (no vim escape sequences)

### Test 8: Buffer After vim
- Action: Refresh page after running vim
- Expected: No VT100 junk in replay
- Expected: Only prompt and normal output

### Test 9-11: Multiple HTML Outputs
- Run `lsh` multiple times
- Expected: Links numbered sequentially (#1, #2, #3...)
- Expected: Each link displays its own HTML content
- Expected: HTML content is clean (no escape sequences)

## Architecture

### Server (main.go)

**PTY Stream Processing:**
```
1. Read PTY output
2. extractAndStoreHTML(data) -> (processedData, widgetIDs)
   - Finds HTML_START...HTML_END blocks
   - Extracts clean HTML content
   - Stores in htmlWidgets map with unique ID
   - Replaces with OSC 8 hyperlink in data
3. Add processedData to buffer (for replay)
4. Broadcast processedData to all clients (terminal output with links)
5. Broadcast HTML notification to live clients (auto-display)
```

**Messages to Clients:**
- Binary: Terminal output (with HTML replaced by links)
- JSON: `{"kind":"status", "state":"waiting|running"}`
- JSON: `{"kind":"html", "widget_id":123}` (live clients only)

### Client (index.html)

**Message Handling:**
- Binary: Write to xterm.js terminal
- JSON status: Update status indicator
- JSON html: Auto-fetch and display HTML widget (live only)

**Link Clicking:**
- xterm.js WebLinksAddon detects `htmlwidget:N` URLs
- Fetches `/htmlwidget/N`
- Displays in HTML panel

## Debugging

### Check HTML Content is Clean
1. Run `lsh`
2. Right-click HTML panel → Inspect Element
3. Look for escape sequences like `\x1b` or `[1m`
4. Should only see clean HTML/CSS

### Check Buffer is Clean
1. Run `vim`, then `:q`
2. Refresh browser
3. Check terminal output
4. Should not see `^[[?1049h` or similar escape sequences

### Check Links Work
1. Run `lsh` 3 times
2. Should see 3 links with IDs #1, #2, #3
3. Click each link
4. Each should display different content (if directory changed)
