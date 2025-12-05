// TreeTable - Keyboard-navigable tree table with multi-selection
//
// Usage:
//   const table = new TreeTable(containerEl, {
//       selectionManager: new StickySelectionManager(),
//       onAction: (items) => { /* insert to terminal */ },
//       onExit: () => { /* return focus to terminal */ }
//   });
//   table.activate();

import { StickySelectionManager } from './selection-manager.js';

export class TreeTable {
    constructor(containerEl, options = {}) {
        this.container = containerEl;
        this.rows = [];           // All rows in DOM order
        this.rowMap = new Map();  // id -> {el, id, value, type, expandable, childrenEl, toggleEl}

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
        // Collect rows from DOM
        const rowEls = this.container.querySelectorAll('.tree-row[data-row-id]');
        rowEls.forEach((el) => {
            const id = el.dataset.rowId;
            const li = el.closest('li');

            // Find associated children container and toggle
            const childrenEl = li ? li.querySelector(':scope > .tree-children') : null;
            const toggleEl = el.querySelector('.tree-toggle:not(.empty)');

            const row = {
                el,
                id,
                value: el.dataset.value || '',
                type: el.dataset.type || 'file',
                expandable: toggleEl !== null,
                childrenEl,
                toggleEl
            };
            this.rows.push(row);
            this.rowMap.set(id, row);
        });

        // Attach listeners
        this.container.addEventListener('keydown', this._handleKeyDown);
        this.container.addEventListener('click', this._handleClick);
    }

    activate() {
        this.active = true;
        this.container.classList.add('active');

        // Set cursor to first visible row if not set
        if (this.selectionManager.getCursor() === null) {
            const visibleRows = this._getVisibleRows();
            if (visibleRows.length > 0) {
                this.selectionManager.onNavigate(visibleRows[0].id);
            }
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
            case 'ArrowDown':
                this._moveCursor(1);
                e.preventDefault();
                break;

            case 'ArrowUp':
                this._moveCursor(-1);
                e.preventDefault();
                break;

            case 'ArrowRight':
                this._expandCurrent();
                e.preventDefault();
                break;

            case 'ArrowLeft':
                this._collapseCurrent();
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
        // Don't interfere with toggle button clicks
        if (e.target.closest('.tree-toggle')) {
            return;
        }

        const rowEl = e.target.closest('.tree-row[data-row-id]');
        if (!rowEl) return;

        const id = rowEl.dataset.rowId;
        if (id === undefined) return;

        // Move cursor to clicked row
        this.selectionManager.onNavigate(id);

        // Toggle selection on click with modifier
        if (e.ctrlKey || e.metaKey || e.shiftKey) {
            this.selectionManager.onToggle(id);
        }
    }

    _getVisibleRows() {
        // Return rows that are not inside collapsed parents
        return this.rows.filter(row => this._isRowVisible(row));
    }

    _isRowVisible(row) {
        // Check if any parent tree-children is collapsed
        let parent = row.el.closest('.tree-children');
        while (parent && parent !== this.container) {
            if (!parent.classList.contains('expanded')) {
                return false;
            }
            parent = parent.parentElement.closest('.tree-children');
        }
        return true;
    }

    _moveCursor(delta) {
        const visibleRows = this._getVisibleRows();
        if (visibleRows.length === 0) return;

        const currentId = this.selectionManager.getCursor();
        const currentIndex = visibleRows.findIndex(row => row.id === currentId);

        let newIndex;
        if (currentIndex === -1) {
            // Cursor not on visible row, go to first or last
            newIndex = delta > 0 ? 0 : visibleRows.length - 1;
        } else {
            newIndex = currentIndex + delta;
            if (newIndex < 0) newIndex = 0;
            if (newIndex >= visibleRows.length) newIndex = visibleRows.length - 1;
        }

        if (newIndex >= 0 && newIndex < visibleRows.length) {
            this.selectionManager.onNavigate(visibleRows[newIndex].id);
            this._scrollRowIntoView(visibleRows[newIndex]);
        }
    }

    _expandCurrent() {
        const currentId = this.selectionManager.getCursor();
        const row = this.rowMap.get(currentId);
        if (!row || !row.expandable || !row.childrenEl) return;

        // Already expanded? Move to first child
        if (row.childrenEl.classList.contains('expanded')) {
            const visibleRows = this._getVisibleRows();
            const currentIndex = visibleRows.findIndex(r => r.id === currentId);
            if (currentIndex >= 0 && currentIndex + 1 < visibleRows.length) {
                this.selectionManager.onNavigate(visibleRows[currentIndex + 1].id);
                this._scrollRowIntoView(visibleRows[currentIndex + 1]);
            }
            return;
        }

        // Expand
        row.childrenEl.classList.add('expanded');
        if (row.toggleEl) {
            row.toggleEl.textContent = '▼';
        }
    }

    _collapseCurrent() {
        const currentId = this.selectionManager.getCursor();
        const row = this.rowMap.get(currentId);
        if (!row) return;

        // If expandable and expanded, collapse it
        if (row.expandable && row.childrenEl && row.childrenEl.classList.contains('expanded')) {
            row.childrenEl.classList.remove('expanded');
            if (row.toggleEl) {
                row.toggleEl.textContent = '▶';
            }
            return;
        }

        // Otherwise, move to parent row
        const parentChildrenEl = row.el.closest('.tree-children');
        if (parentChildrenEl) {
            const parentLi = parentChildrenEl.closest('li');
            if (parentLi) {
                const parentRowEl = parentLi.querySelector(':scope > .tree-row[data-row-id]');
                if (parentRowEl) {
                    const parentId = parentRowEl.dataset.rowId;
                    this.selectionManager.onNavigate(parentId);
                    const parentRow = this.rowMap.get(parentId);
                    if (parentRow) {
                        this._scrollRowIntoView(parentRow);
                    }
                }
            }
        }
    }

    _scrollRowIntoView(row) {
        row.el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }

    _selectAll() {
        const visibleRows = this._getVisibleRows();
        const allIds = visibleRows.map(row => row.id);
        this.selectionManager.selectAll(allIds);
    }

    _updateVisuals() {
        const selected = this.selectionManager.getSelection();
        const cursor = this.selectionManager.getCursor();

        this.rows.forEach(row => {
            row.el.classList.toggle('selected', selected.has(row.id));
            row.el.classList.toggle('cursor', row.id === cursor && this.active);
        });
    }

    /**
     * Get items for action - selected items, or cursor if nothing selected
     */
    getSelectedItems() {
        const effectiveSelection = this.selectionManager.getEffectiveSelection();
        return this.rows.filter(row => effectiveSelection.has(row.id));
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
            this._flashCopyFeedback();
            if (this.onCopy) {
                this.onCopy(items);
            }
        } catch (err) {
            console.error('Failed to copy to clipboard:', err);
        }
    }

    _flashCopyFeedback() {
        const items = this.getSelectedItems();
        items.forEach(item => {
            item.el.classList.add('copy-flash');
            setTimeout(() => {
                item.el.classList.remove('copy-flash');
            }, 200);
        });
    }
}
