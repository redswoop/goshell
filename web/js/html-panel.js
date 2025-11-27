// HTML output panel management

import * as splitter from './splitter.js';

let panelEl = null;
let toggleBtnEl = null;
let resizeCallback = null;

export function init(panel, splitterEl, toggleBtn, options = {}) {
    panelEl = panel;
    toggleBtnEl = toggleBtn;

    if (options.onResize) {
        resizeCallback = options.onResize;
    }

    // Initialize splitter
    splitter.init(splitterEl, panelEl, {
        minHeight: 100,
        maxHeight: () => window.innerHeight - 200,
        onDrag: () => {
            if (resizeCallback) resizeCallback();
        }
    });

    // Toggle button click handler
    toggleBtnEl.addEventListener('click', toggle);
}

export function show(html, animate = true) {
    if (html !== undefined) {
        panelEl.innerHTML = html;
    }

    if (animate) {
        panelEl.classList.add('animating');
    }

    panelEl.classList.add('visible');
    splitter.show();
    toggleBtnEl.textContent = 'Hide HTML';

    if (animate) {
        setTimeout(() => {
            panelEl.classList.remove('animating');
            if (resizeCallback) resizeCallback();
        }, 350);
    } else {
        if (resizeCallback) resizeCallback();
    }
}

export function hide(animate = true) {
    if (animate) {
        panelEl.classList.add('animating');
    }

    panelEl.classList.remove('visible');
    splitter.hide();
    toggleBtnEl.textContent = 'Show HTML';

    if (animate) {
        setTimeout(() => {
            panelEl.classList.remove('animating');
            if (resizeCallback) resizeCallback();
        }, 350);
    } else {
        if (resizeCallback) resizeCallback();
    }
}

export function toggle() {
    if (panelEl.classList.contains('visible')) {
        hide();
    } else {
        show();
    }
}

export function isVisible() {
    return panelEl.classList.contains('visible');
}

export async function loadWidget(widgetId) {
    try {
        const response = await fetch(`/htmlwidget/${widgetId}`);
        const html = await response.text();
        show(html);
    } catch (err) {
        console.error('Failed to load HTML widget:', err);
    }
}

export function onResize(callback) {
    resizeCallback = callback;
}
