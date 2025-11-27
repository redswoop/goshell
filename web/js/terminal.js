// Terminal module - wraps xterm.js

let term = null;
let fitAddon = null;
let linkActivateCallback = null;
let customKeyHandlers = [];  // Array of {key, ctrl, handler} objects

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

    // Use window-level capturing to intercept keys before anything else
    window.addEventListener('keydown', (event) => {
        for (const handler of customKeyHandlers) {
            const matches = event.code === handler.code &&
                event.ctrlKey === !!handler.ctrl &&
                event.shiftKey === !!handler.shift &&
                event.altKey === !!handler.alt &&
                event.metaKey === !!handler.meta;

            if (matches) {
                event.preventDefault();
                event.stopPropagation();
                event.stopImmediatePropagation();
                handler.callback(event);
                return;
            }
        }
    }, true);  // true = capturing phase, fires first

    return term;
}

// Register a custom key handler that intercepts before xterm
// Options: { code: 'Space', ctrl: true, shift: false, callback: (e) => {} }
export function registerKeyHandler(options) {
    customKeyHandlers.push(options);
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
