// HTML output panel management

import * as splitter from './splitter.js';
import { TokenGrid } from './token-grid.js';
import { StickySelectionManager } from './selection-manager.js';

let panelEl = null;
let toggleBtnEl = null;
let resizeCallback = null;
let exitCallback = null;      // Called when user exits panel focus
let actionCallback = null;    // Called when user performs action (e.g., insert to terminal)
let activeGrid = null;        // Current TokenGrid instance

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
        // Initialize grid after content is set
        initializeGrid();
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

// Set callback for when user exits panel focus (returns to terminal)
export function setExitCallback(callback) {
    exitCallback = callback;
}

// Set callback for when user performs an action (e.g., insert items to terminal)
export function setActionCallback(callback) {
    actionCallback = callback;
}

// Initialize TokenGrid if navigable content is present
function initializeGrid() {
    // Destroy previous grid if any
    if (activeGrid) {
        activeGrid.destroy();
        activeGrid = null;
    }

    // Look for token grid container
    const gridEl = panelEl.querySelector('[data-grid-id]');
    if (!gridEl) return;

    activeGrid = new TokenGrid(gridEl, {
        selectionManager: new StickySelectionManager(),
        onAction: handleGridAction,
        onExit: handleGridExit,
        onCopy: handleGridCopy
    });
}

function handleGridAction(items) {
    // Insert items to terminal
    const text = items.map(item => item.value).join(' ');
    if (actionCallback) {
        actionCallback(text);
    }
    // Return focus to terminal after action
    handleGridExit();
}

function handleGridExit() {
    if (activeGrid) {
        activeGrid.deactivate();
    }
    if (exitCallback) {
        exitCallback();
    }
}

function handleGridCopy(items) {
    // Optional: could show notification
    console.log(`Copied ${items.length} item(s) to clipboard`);
}

// Enter focus mode for the HTML panel (called via hotkey)
export function enterFocus() {
    if (!isVisible()) {
        return false;
    }
    if (activeGrid) {
        activeGrid.activate();
        return true;
    }
    return false;
}

// Check if panel has a navigable grid
export function hasGrid() {
    return activeGrid !== null;
}
