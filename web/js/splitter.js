// Drag splitter functionality for resizing panes

let isDragging = false;
let startY = 0;
let startHeight = 0;
let targetEl = null;
let splitterEl = null;
let onDragCallback = null;
let minHeight = 100;
let maxHeightFn = () => window.innerHeight - 200;

export function init(splitter, target, options = {}) {
    splitterEl = splitter;
    targetEl = target;

    if (options.minHeight !== undefined) minHeight = options.minHeight;
    if (options.maxHeight !== undefined) maxHeightFn = typeof options.maxHeight === 'function'
        ? options.maxHeight
        : () => options.maxHeight;
    if (options.onDrag) onDragCallback = options.onDrag;

    splitterEl.addEventListener('mousedown', handleMouseDown);
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
}

function handleMouseDown(e) {
    isDragging = true;
    startY = e.clientY;
    startHeight = targetEl.offsetHeight;
    splitterEl.classList.add('dragging');
    document.body.style.cursor = 'ns-resize';
    document.body.style.userSelect = 'none';
    e.preventDefault();
}

function handleMouseMove(e) {
    if (!isDragging) return;

    const delta = e.clientY - startY;
    const maxHeight = maxHeightFn();
    const newHeight = Math.max(minHeight, Math.min(maxHeight, startHeight + delta));

    targetEl.style.height = newHeight + 'px';

    if (onDragCallback) {
        onDragCallback(newHeight);
    }
}

function handleMouseUp() {
    if (isDragging) {
        isDragging = false;
        splitterEl.classList.remove('dragging');
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    }
}

export function show() {
    splitterEl.classList.add('visible');
}

export function hide() {
    splitterEl.classList.remove('visible');
}

export function destroy() {
    splitterEl.removeEventListener('mousedown', handleMouseDown);
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
}
