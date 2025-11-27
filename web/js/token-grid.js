// TokenGrid - Keyboard-navigable grid with multi-selection
//
// Usage:
//   const grid = new TokenGrid(containerEl, {
//       selectionManager: new StickySelectionManager(),
//       onAction: (items) => { /* insert to terminal */ },
//       onExit: () => { /* return focus to terminal */ }
//   });
//   grid.activate();

import { StickySelectionManager } from './selection-manager.js';

export class TokenGrid {
    constructor(containerEl, options = {}) {
        this.container = containerEl;
        this.items = [];
        this.itemMap = new Map(); // id -> {el, id, value, type}

        // Callbacks
        this.onAction = options.onAction || null;  // (items) => void
        this.onExit = options.onExit || null;      // () => void
        this.onCopy = options.onCopy || null;      // (items) => void

        // Selection manager (pluggable)
        this.selectionManager = options.selectionManager || new StickySelectionManager();
        this.selectionManager.onChange = () => this._updateVisuals();

        // State
        this.active = false;

        // Bound handlers for cleanup
        this._handleKeyDown = this._handleKeyDown.bind(this);
        this._handleClick = this._handleClick.bind(this);

        this._init();
    }

    _init() {
        // Collect items from DOM
        const itemEls = this.container.querySelectorAll('.token-item');
        itemEls.forEach((el, index) => {
            const id = el.dataset.id || String(index);
            const item = {
                el,
                id,
                value: el.dataset.value || '',
                type: el.dataset.type || 'file'
            };
            this.items.push(item);
            this.itemMap.set(id, item);
        });

        // Attach listeners
        this.container.addEventListener('keydown', this._handleKeyDown);
        this.container.addEventListener('click', this._handleClick);
    }

    activate() {
        this.active = true;
        this.container.classList.add('active');

        // Set cursor to first item if not set
        if (this.selectionManager.getCursor() === null && this.items.length > 0) {
            this.selectionManager.onNavigate(this.items[0].id);
        }

        this.container.focus();
        this._updateVisuals();
    }

    deactivate() {
        this.active = false;
        this.container.classList.remove('active');
        this._updateVisuals();
    }

    destroy() {
        this.container.removeEventListener('keydown', this._handleKeyDown);
        this.container.removeEventListener('click', this._handleClick);
        this.selectionManager.clear();
        this.deactivate();
    }

    _handleKeyDown(e) {
        switch (e.key) {
            case 'ArrowRight':
                this._moveCursor(1);
                e.preventDefault();
                break;

            case 'ArrowLeft':
                this._moveCursor(-1);
                e.preventDefault();
                break;

            case 'ArrowDown':
                this._moveCursor(this._getColumnsCount());
                e.preventDefault();
                break;

            case 'ArrowUp':
                this._moveCursor(-this._getColumnsCount());
                e.preventDefault();
                break;

            case ' ': // Spacebar - toggle selection
                this.selectionManager.onToggle();
                e.preventDefault();
                break;

            case 'Enter': // Primary action - insert to terminal
                this._performAction();
                e.preventDefault();
                break;

            case 'Escape':
                if (this.onExit) {
                    this.onExit();
                }
                e.preventDefault();
                break;

            case 'a':
                if (e.ctrlKey || e.metaKey) {
                    this._selectAll();
                    e.preventDefault();
                }
                break;

            case 'c':
                if (e.ctrlKey || e.metaKey) {
                    this._copyToClipboard();
                    e.preventDefault();
                }
                break;
        }
    }

    _handleClick(e) {
        const itemEl = e.target.closest('.token-item');
        if (!itemEl) return;

        const id = itemEl.dataset.id;
        if (id === undefined) return;

        // Move cursor to clicked item
        this.selectionManager.onNavigate(id);

        // Toggle selection on click (sticky mode behavior)
        if (e.ctrlKey || e.metaKey || e.shiftKey) {
            this.selectionManager.onToggle(id);
        }
    }

    _moveCursor(delta) {
        const currentId = this.selectionManager.getCursor();
        const currentIndex = this.items.findIndex(item => item.id === currentId);

        let newIndex = currentIndex + delta;
        if (newIndex < 0) newIndex = 0;
        if (newIndex >= this.items.length) newIndex = this.items.length - 1;

        if (newIndex !== currentIndex && newIndex >= 0 && newIndex < this.items.length) {
            this.selectionManager.onNavigate(this.items[newIndex].id);
            this._scrollItemIntoView(this.items[newIndex]);
        }
    }

    _getColumnsCount() {
        // Calculate columns based on item positions in flexbox
        if (this.items.length < 2) return 1;

        const firstTop = this.items[0].el.offsetTop;
        for (let i = 1; i < this.items.length; i++) {
            if (this.items[i].el.offsetTop !== firstTop) {
                return i;
            }
        }
        return this.items.length; // All on one row
    }

    _scrollItemIntoView(item) {
        item.el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }

    _selectAll() {
        const allIds = this.items.map(item => item.id);
        this.selectionManager.selectAll(allIds);
    }

    _updateVisuals() {
        const selected = this.selectionManager.getSelection();
        const cursor = this.selectionManager.getCursor();

        this.items.forEach(item => {
            item.el.classList.toggle('selected', selected.has(item.id));
            item.el.classList.toggle('cursor', item.id === cursor && this.active);
        });
    }

    /**
     * Get items for action - selected items, or cursor if nothing selected
     */
    getSelectedItems() {
        const effectiveSelection = this.selectionManager.getEffectiveSelection();
        return this.items.filter(item => effectiveSelection.has(item.id));
    }

    _performAction() {
        const items = this.getSelectedItems();
        if (items.length > 0 && this.onAction) {
            this.onAction(items);
        }
    }

    async _copyToClipboard() {
        const items = this.getSelectedItems();
        if (items.length === 0) return;

        const text = items.map(item => item.value).join(' ');
        try {
            await navigator.clipboard.writeText(text);
            // Brief visual feedback
            this._flashCopyFeedback();
            if (this.onCopy) {
                this.onCopy(items);
            }
        } catch (err) {
            console.error('Failed to copy to clipboard:', err);
        }
    }

    _flashCopyFeedback() {
        // Brief flash on selected/cursor items
        const items = this.getSelectedItems();
        items.forEach(item => {
            item.el.classList.add('copy-flash');
            setTimeout(() => {
                item.el.classList.remove('copy-flash');
            }, 200);
        });
    }
}
