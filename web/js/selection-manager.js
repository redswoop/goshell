// Selection Manager - Pluggable selection strategies for TokenGrid
//
// Interface:
//   getSelection()   - Returns Set of selected item IDs
//   getCursor()      - Returns current cursor ID (may not be selected)
//   clear()          - Clear all selections and cursor
//   onNavigate(id)   - Called when cursor moves to a new item
//   onToggle(id)     - Called when user toggles selection (spacebar)
//   selectAll(ids)   - Select all provided IDs
//   onChange         - Callback when selection state changes

/**
 * Base SelectionManager class - defines the interface
 */
export class SelectionManager {
    constructor() {
        this.onChange = null; // Callback: () => void
    }

    getSelection() {
        throw new Error('getSelection() must be implemented');
    }

    getCursor() {
        throw new Error('getCursor() must be implemented');
    }

    clear() {
        throw new Error('clear() must be implemented');
    }

    onNavigate(id) {
        throw new Error('onNavigate() must be implemented');
    }

    onToggle(id) {
        throw new Error('onToggle() must be implemented');
    }

    selectAll(ids) {
        throw new Error('selectAll() must be implemented');
    }

    _notifyChange() {
        if (this.onChange) {
            this.onChange();
        }
    }
}

/**
 * StickySelectionManager - Multiple selection mode
 *
 * Behavior:
 * - Navigation (arrow keys) moves cursor WITHOUT changing selection
 * - Spacebar toggles the item at cursor in/out of selection
 * - Selection persists across navigation
 * - If nothing is explicitly selected, cursor item is implicit selection
 */
export class StickySelectionManager extends SelectionManager {
    constructor() {
        super();
        this.selected = new Set();
        this.cursor = null;
    }

    getSelection() {
        return new Set(this.selected);
    }

    getCursor() {
        return this.cursor;
    }

    clear() {
        this.selected.clear();
        this.cursor = null;
        this._notifyChange();
    }

    onNavigate(id) {
        this.cursor = id;
        this._notifyChange();
    }

    onToggle(id) {
        if (id === null || id === undefined) {
            id = this.cursor;
        }
        if (id === null || id === undefined) {
            return;
        }

        if (this.selected.has(id)) {
            this.selected.delete(id);
        } else {
            this.selected.add(id);
        }
        this._notifyChange();
    }

    selectAll(ids) {
        for (const id of ids) {
            this.selected.add(id);
        }
        this._notifyChange();
    }

    /**
     * Get effective selection - if nothing selected, return cursor item
     */
    getEffectiveSelection() {
        if (this.selected.size > 0) {
            return new Set(this.selected);
        }
        if (this.cursor !== null) {
            return new Set([this.cursor]);
        }
        return new Set();
    }
}
