/**
 * HeadlessTerm - JavaScript wrapper for the Go WASM terminal emulator
 *
 * This provides a clean ES6 class interface with event emitters for all
 * terminal events (bell, title changes, clipboard, etc.)
 */

class Terminal {
    /**
     * Create a new terminal instance
     * @param {number} rows - Number of rows (default: 24)
     * @param {number} cols - Number of columns (default: 80)
     */
    constructor(rows = 24, cols = 80) {
        if (!window.HeadlessTerm) {
            throw new Error('HeadlessTerm WASM module not loaded. Call Terminal.init() first.');
        }

        this._id = window.HeadlessTerm.create(rows, cols);
        this._eventHandlers = {
            bell: [],
            title: [],
            clipboard: [],
            notification: [],
            ptyWrite: [],
            size: [],
            semanticPrompt: [],
            apc: [],
            pm: [],
            sos: [],
            scrollback: [],
            recording: []
        };

        // Register internal handlers
        this._setupHandlers();
    }

    /**
     * Initialize the WASM module
     * @returns {Promise<void>}
     */
    static async init(wasmPath = 'headlessterm.wasm') {
        if (window.HeadlessTerm) {
            return; // Already initialized
        }

        const go = new Go();
        const result = await WebAssembly.instantiateStreaming(
            fetch(wasmPath),
            go.importObject
        );

        go.run(result.instance);

        // Wait for HeadlessTerm to be available
        await new Promise((resolve) => {
            const check = () => {
                if (window.HeadlessTerm) {
                    resolve();
                } else {
                    setTimeout(check, 10);
                }
            };
            check();
        });
    }

    _setupHandlers() {
        const HT = window.HeadlessTerm;

        // Bell handler
        HT.onBell(this._id, () => {
            this._emit('bell');
        });

        // Title handler
        HT.onTitle(this._id, (event, title) => {
            this._emit('title', { event, title });
        });

        // Clipboard handler
        HT.onClipboard(this._id, (event, clipboard, data) => {
            if (event === 'read') {
                // Return data from handlers (first non-empty wins)
                for (const handler of this._eventHandlers.clipboard) {
                    const result = handler({ event, clipboard });
                    if (result) return result;
                }
                return '';
            } else {
                this._emit('clipboard', { event, clipboard, data });
            }
        });

        // Notification handler
        HT.onNotification(this._id, (payload) => {
            this._emit('notification', payload);
            return '';
        });

        // PTY write handler
        HT.onPTYWrite(this._id, (data) => {
            this._emit('ptyWrite', data);
        });

        // Size handler
        HT.onSize(this._id, (query) => {
            for (const handler of this._eventHandlers.size) {
                const result = handler(query);
                if (result) return result;
            }
            // Default sizes
            return query === 'window'
                ? { width: 800, height: 600 }
                : { width: 10, height: 20 };
        });

        // Semantic prompt handler
        HT.onSemanticPrompt(this._id, (mark, exitCode) => {
            this._emit('semanticPrompt', { mark, exitCode });
        });

        // APC handler
        HT.onAPC(this._id, (data) => {
            this._emit('apc', data);
        });

        // PM handler
        HT.onPM(this._id, (data) => {
            this._emit('pm', data);
        });

        // SOS handler
        HT.onSOS(this._id, (data) => {
            this._emit('sos', data);
        });

        // Scrollback handler
        HT.onScrollback(this._id, (event, data) => {
            for (const handler of this._eventHandlers.scrollback) {
                const result = handler(event, data);
                if (result !== undefined) return result;
            }
            return null;
        });

        // Recording handler
        HT.onRecording(this._id, (event, data) => {
            for (const handler of this._eventHandlers.recording) {
                const result = handler(event, data);
                if (result !== undefined) return result;
            }
            return null;
        });
    }

    _emit(event, data) {
        for (const handler of this._eventHandlers[event]) {
            handler(data);
        }
    }

    /**
     * Register an event handler
     * @param {string} event - Event name
     * @param {Function} handler - Handler function
     */
    on(event, handler) {
        if (this._eventHandlers[event]) {
            this._eventHandlers[event].push(handler);
        }
        return this;
    }

    /**
     * Remove an event handler
     * @param {string} event - Event name
     * @param {Function} handler - Handler function
     */
    off(event, handler) {
        if (this._eventHandlers[event]) {
            const index = this._eventHandlers[event].indexOf(handler);
            if (index !== -1) {
                this._eventHandlers[event].splice(index, 1);
            }
        }
        return this;
    }

    /**
     * Destroy the terminal instance
     */
    destroy() {
        window.HeadlessTerm.destroy(this._id);
        this._id = null;
    }

    // ========================================================================
    // Input Processing
    // ========================================================================

    /**
     * Write raw bytes to the terminal
     * @param {Uint8Array} data - Bytes to write
     * @returns {number} Number of bytes written
     */
    write(data) {
        if (typeof data === 'string') {
            data = new TextEncoder().encode(data);
        }
        return window.HeadlessTerm.write(this._id, data);
    }

    /**
     * Write a string to the terminal
     * @param {string} str - String to write
     * @returns {number} Number of bytes written
     */
    writeString(str) {
        return window.HeadlessTerm.writeString(this._id, str);
    }

    // ========================================================================
    // Dimensions
    // ========================================================================

    /**
     * Resize the terminal
     * @param {number} rows - New number of rows
     * @param {number} cols - New number of columns
     */
    resize(rows, cols) {
        window.HeadlessTerm.resize(this._id, rows, cols);
    }

    /**
     * Get the number of rows
     * @returns {number}
     */
    get rows() {
        return window.HeadlessTerm.rows(this._id);
    }

    /**
     * Get the number of columns
     * @returns {number}
     */
    get cols() {
        return window.HeadlessTerm.cols(this._id);
    }

    // ========================================================================
    // Cursor
    // ========================================================================

    /**
     * Get the cursor position
     * @returns {{row: number, col: number}}
     */
    get cursorPos() {
        return window.HeadlessTerm.cursorPos(this._id);
    }

    /**
     * Check if cursor is visible
     * @returns {boolean}
     */
    get cursorVisible() {
        return window.HeadlessTerm.cursorVisible(this._id);
    }

    /**
     * Get cursor style
     * @returns {number} 0=BlinkingBlock, 1=SteadyBlock, 2=BlinkingUnderline, etc.
     */
    get cursorStyle() {
        return window.HeadlessTerm.cursorStyle(this._id);
    }

    // ========================================================================
    // Content
    // ========================================================================

    /**
     * Get the visible screen content as a string
     * @returns {string}
     */
    getString() {
        return window.HeadlessTerm.getString(this._id);
    }

    /**
     * Get the content of a specific line
     * @param {number} row - Row index (0-based)
     * @returns {string}
     */
    lineContent(row) {
        return window.HeadlessTerm.lineContent(this._id, row);
    }

    /**
     * Get a specific cell
     * @param {number} row - Row index
     * @param {number} col - Column index
     * @returns {Object} Cell data with char, fg, bg, and attribute flags
     */
    cell(row, col) {
        return window.HeadlessTerm.cell(this._id, row, col);
    }

    /**
     * Get a snapshot of the terminal state
     * @param {string} detail - "text", "styled", or "full"
     * @returns {Object}
     */
    snapshot(detail = 'styled') {
        return window.HeadlessTerm.snapshot(this._id, detail);
    }

    /**
     * Get a snapshot as JSON string
     * @param {string} detail - "text", "styled", or "full"
     * @returns {string}
     */
    snapshotJSON(detail = 'styled') {
        return window.HeadlessTerm.snapshotJSON(this._id, detail);
    }

    // ========================================================================
    // State Inspection
    // ========================================================================

    /**
     * Get the window title
     * @returns {string}
     */
    get title() {
        return window.HeadlessTerm.title(this._id);
    }

    /**
     * Check if a terminal mode is enabled
     * @param {number} mode - Mode flag
     * @returns {boolean}
     */
    hasMode(mode) {
        return window.HeadlessTerm.hasMode(this._id, mode);
    }

    /**
     * Check if alternate screen is active
     * @returns {boolean}
     */
    get isAlternateScreen() {
        return window.HeadlessTerm.isAlternateScreen(this._id);
    }

    /**
     * Get the scroll region
     * @returns {{top: number, bottom: number}}
     */
    get scrollRegion() {
        return window.HeadlessTerm.scrollRegion(this._id);
    }

    // ========================================================================
    // Scrollback
    // ========================================================================

    /**
     * Get the number of scrollback lines
     * @returns {number}
     */
    get scrollbackLen() {
        return window.HeadlessTerm.scrollbackLen(this._id);
    }

    /**
     * Get a scrollback line
     * @param {number} index - Line index (0 is oldest)
     * @returns {Array<Object>} Array of cells
     */
    scrollbackLine(index) {
        return window.HeadlessTerm.scrollbackLine(this._id, index);
    }

    /**
     * Clear all scrollback
     */
    clearScrollback() {
        window.HeadlessTerm.clearScrollback(this._id);
    }

    // ========================================================================
    // Selection
    // ========================================================================

    /**
     * Set the text selection
     * @param {number} startRow
     * @param {number} startCol
     * @param {number} endRow
     * @param {number} endCol
     */
    setSelection(startRow, startCol, endRow, endCol) {
        window.HeadlessTerm.setSelection(this._id, startRow, startCol, endRow, endCol);
    }

    /**
     * Clear the text selection
     */
    clearSelection() {
        window.HeadlessTerm.clearSelection(this._id);
    }

    /**
     * Check if there's an active selection
     * @returns {boolean}
     */
    get hasSelection() {
        return window.HeadlessTerm.hasSelection(this._id);
    }

    /**
     * Get the selected text
     * @returns {string}
     */
    getSelectedText() {
        return window.HeadlessTerm.getSelectedText(this._id);
    }

    // ========================================================================
    // Dirty Tracking
    // ========================================================================

    /**
     * Check if any cells have been modified
     * @returns {boolean}
     */
    get hasDirty() {
        return window.HeadlessTerm.hasDirty(this._id);
    }

    /**
     * Get list of modified cell positions
     * @returns {Array<{row: number, col: number}>}
     */
    dirtyCells() {
        return window.HeadlessTerm.dirtyCells(this._id);
    }

    /**
     * Clear dirty flags
     */
    clearDirty() {
        window.HeadlessTerm.clearDirty(this._id);
    }

    // ========================================================================
    // Search
    // ========================================================================

    /**
     * Search for a pattern in the visible screen
     * @param {string} pattern - Search pattern
     * @returns {Array<{row: number, col: number}>}
     */
    search(pattern) {
        return window.HeadlessTerm.search(this._id, pattern);
    }

    /**
     * Search for a pattern in scrollback
     * @param {string} pattern - Search pattern
     * @returns {Array<{row: number, col: number}>}
     */
    searchScrollback(pattern) {
        return window.HeadlessTerm.searchScrollback(this._id, pattern);
    }

    // ========================================================================
    // Working Directory & User Vars
    // ========================================================================

    /**
     * Get the working directory URI
     * @returns {string}
     */
    get workingDirectory() {
        return window.HeadlessTerm.workingDirectory(this._id);
    }

    /**
     * Get the working directory path (extracted from URI)
     * @returns {string}
     */
    get workingDirectoryPath() {
        return window.HeadlessTerm.workingDirectoryPath(this._id);
    }

    /**
     * Get a user variable
     * @param {string} name - Variable name
     * @returns {string}
     */
    getUserVar(name) {
        return window.HeadlessTerm.getUserVar(this._id, name);
    }

    /**
     * Get all user variables
     * @returns {Object}
     */
    getUserVars() {
        return window.HeadlessTerm.getUserVars(this._id);
    }

    // ========================================================================
    // Semantic Prompts
    // ========================================================================

    /**
     * Get all prompt marks
     * @returns {Array<{type: number, row: number, exitCode: number}>}
     */
    get promptMarks() {
        return window.HeadlessTerm.promptMarks(this._id);
    }

    /**
     * Get the last command output
     * @returns {string}
     */
    getLastCommandOutput() {
        return window.HeadlessTerm.getLastCommandOutput(this._id);
    }

    // ========================================================================
    // Images
    // ========================================================================

    /**
     * Get the number of stored images
     * @returns {number}
     */
    get imageCount() {
        return window.HeadlessTerm.imageCount(this._id);
    }

    /**
     * Get the number of image placements
     * @returns {number}
     */
    get imagePlacementCount() {
        return window.HeadlessTerm.imagePlacementCount(this._id);
    }

    /**
     * Get image memory usage in bytes
     * @returns {number}
     */
    get imageUsedMemory() {
        return window.HeadlessTerm.imageUsedMemory(this._id);
    }

    /**
     * Check if Sixel graphics is enabled
     * @returns {boolean}
     */
    get sixelEnabled() {
        return window.HeadlessTerm.sixelEnabled(this._id);
    }

    /**
     * Check if Kitty graphics is enabled
     * @returns {boolean}
     */
    get kittyEnabled() {
        return window.HeadlessTerm.kittyEnabled(this._id);
    }
}

// Terminal mode constants
Terminal.Mode = {
    CursorKeys: 1 << 0,
    ColumnMode: 1 << 1,
    Insert: 1 << 2,
    Origin: 1 << 3,
    LineWrap: 1 << 4,
    BlinkingCursor: 1 << 5,
    LineFeedNewLine: 1 << 6,
    ShowCursor: 1 << 7,
    ReportMouseClicks: 1 << 8,
    ReportCellMouseMotion: 1 << 9,
    ReportAllMouseMotion: 1 << 10,
    ReportFocusInOut: 1 << 11,
    UTF8Mouse: 1 << 12,
    SGRMouse: 1 << 13,
    AlternateScroll: 1 << 14,
    UrgencyHints: 1 << 15,
    SwapScreenAndSetRestoreCursor: 1 << 16,
    BracketedPaste: 1 << 17,
    KeypadApplication: 1 << 18
};

// Cursor style constants
Terminal.CursorStyle = {
    BlinkingBlock: 0,
    SteadyBlock: 1,
    BlinkingUnderline: 2,
    SteadyUnderline: 3,
    BlinkingBar: 4,
    SteadyBar: 5
};

// Semantic prompt mark types
Terminal.PromptMark = {
    PromptStart: 0,
    PromptEnd: 1,
    CommandStart: 2,
    CommandEnd: 3
};

// Export for ES modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = Terminal;
}
