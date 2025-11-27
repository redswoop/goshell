// Terminal module - wraps xterm.js

let term = null;
let fitAddon = null;
let linkActivateCallback = null;

export function init(containerEl, options = {}) {
    if (options.onLinkActivate) {
        linkActivateCallback = options.onLinkActivate;
    }

    term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        theme: {
            background: '#1e1e1e',
            foreground: '#d4d4d4'
        },
        linkHandler: {
            activate: (event, uri) => {
                if (linkActivateCallback) {
                    const handled = linkActivateCallback(uri);
                    if (handled) return;
                }
                // Default behavior for http/https
                window.open(uri, '_blank');
            },
            allowNonHttpProtocols: true
        }
    });

    fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);
    term.open(containerEl);

    return term;
}

export function fit() {
    if (fitAddon) {
        fitAddon.fit();
    }
}

export function write(data) {
    if (term) {
        term.write(data);
    }
}

export function clear() {
    if (term) {
        term.clear();
    }
}

export function focus() {
    if (term) {
        term.focus();
    }
}

export function onData(callback) {
    if (term) {
        term.onData(callback);
    }
}

export function getSize() {
    if (term && term.rows && term.cols) {
        return { rows: term.rows, cols: term.cols };
    }
    return null;
}
