// Main entry point - wires all modules together

import * as terminal from './terminal.js';
import * as connection from './connection.js';
import * as htmlPanel from './html-panel.js';
import * as api from './api.js';

// Fit terminal and send size to backend
function fitAndResize() {
    terminal.fit();
    setTimeout(() => {
        const size = terminal.getSize();
        if (size) {
            api.resize(size.rows, size.cols);
        }
    }, 0);
}

// Handle htmlwidget: links
function handleLink(uri) {
    if (uri.startsWith('htmlwidget:')) {
        const widgetId = uri.substring('htmlwidget:'.length);
        htmlPanel.loadWidget(widgetId);
        return true;
    }
    return false;
}

// Initialize everything
function init() {
    const terminalEl = document.getElementById('terminal');
    const htmlOutputEl = document.getElementById('html-output');
    const splitterEl = document.getElementById('splitter');
    const toggleBtn = document.getElementById('toggle-html-btn');
    const restartBtn = document.getElementById('restart-btn');
    const statusEl = document.getElementById('status');

    // Initialize terminal
    terminal.init(terminalEl, {
        onLinkActivate: handleLink
    });

    // Initialize HTML panel with splitter
    htmlPanel.init(htmlOutputEl, splitterEl, toggleBtn, {
        onResize: fitAndResize
    });

    // Initial fit
    fitAndResize();
    terminal.focus();

    // Handle window resize
    window.addEventListener('resize', fitAndResize);

    // Connect WebSocket
    connection.connect('ws://127.0.0.1:7777/ws/shell');

    // Handle terminal output
    connection.onBinary((data) => {
        terminal.write(data);
    });

    // Handle status updates
    connection.onStatus((state) => {
        statusEl.textContent = state;
    });

    // Handle HTML notifications
    connection.onHtml((widgetId) => {
        htmlPanel.loadWidget(widgetId);
    });

    // Handle connection errors
    connection.onError(() => {
        terminal.write('\r\n\x1b[31mWebSocket connection error\x1b[0m\r\n');
    });

    connection.onClose(() => {
        terminal.write('\r\n\x1b[31mConnection closed\x1b[0m\r\n');
    });

    // Send terminal input to WebSocket
    terminal.onData((data) => {
        connection.send(data);
    });

    // Restart button
    restartBtn.addEventListener('click', async () => {
        const success = await api.restart();
        if (success) {
            terminal.clear();
            fitAndResize();
            console.log('Shell restarted');
        } else {
            terminal.write('\r\n\x1b[31mFailed to restart shell\x1b[0m\r\n');
        }
    });

    // Expose runCommand globally for HTML widgets
    window.runCommand = api.runCommand;
}

// Start when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
