// WebSocket connection management

let ws = null;
let binaryCallback = null;
let statusCallback = null;
let htmlCallback = null;
let errorCallback = null;
let closeCallback = null;

export function connect(url) {
    ws = new WebSocket(url);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
        console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
        if (typeof event.data === 'string') {
            // Text message - status update or HTML notification
            try {
                const msg = JSON.parse(event.data);
                if (msg.kind === 'status' && statusCallback) {
                    statusCallback(msg.state);
                } else if (msg.kind === 'html' && htmlCallback) {
                    htmlCallback(msg.widget_id);
                }
            } catch (e) {
                console.error('Failed to parse message:', e);
            }
        } else {
            // Binary message - terminal output
            if (binaryCallback) {
                const data = new Uint8Array(event.data);
                binaryCallback(data);
            }
        }
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        if (errorCallback) {
            errorCallback(error);
        }
    };

    ws.onclose = () => {
        console.log('WebSocket disconnected');
        if (closeCallback) {
            closeCallback();
        }
    };

    return ws;
}

export function send(data) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(data);
    }
}

export function onBinary(callback) {
    binaryCallback = callback;
}

export function onStatus(callback) {
    statusCallback = callback;
}

export function onHtml(callback) {
    htmlCallback = callback;
}

export function onError(callback) {
    errorCallback = callback;
}

export function onClose(callback) {
    closeCallback = callback;
}

export function isOpen() {
    return ws && ws.readyState === WebSocket.OPEN;
}
