# HeadlessTerm WASM

WebAssembly build of the go-headless-term library for use in web browsers.

All terminal handlers (bell, title, clipboard, etc.) are exposed as external JavaScript callbacks, allowing full integration with your web application.

## Building

```bash
# Build everything (copies wasm_exec.js and builds WASM)
make all

# Or step by step:
make setup  # Copy wasm_exec.js from Go installation
make build  # Build the WASM binary
```

## Running the Example

```bash
make serve
# Open http://localhost:8080 in your browser
```

## Usage

### 1. Include Required Scripts

```html
<script src="wasm_exec.js"></script>
<script src="terminal.js"></script>
```

### 2. Initialize and Create Terminal

```javascript
// Initialize the WASM module (only once)
await Terminal.init('headlessterm.wasm');

// Create a terminal instance (24 rows x 80 cols)
const term = new Terminal(24, 80);
```

### 3. Write Data to Terminal

```javascript
// Write a string (supports ANSI escape sequences)
term.writeString('Hello, \x1b[32mWorld\x1b[0m!\r\n');

// Write raw bytes
const data = new Uint8Array([0x1b, 0x5b, 0x32, 0x4a]); // ESC[2J (clear screen)
term.write(data);
```

### 4. Read Terminal State

```javascript
// Get visible content as string
const content = term.getString();

// Get a specific cell
const cell = term.cell(0, 0);
console.log(cell.char, cell.fg, cell.bg, cell.bold);

// Get cursor position
const { row, col } = term.cursorPos;

// Get terminal title
const title = term.title;

// Get snapshot (for rendering)
const snapshot = term.snapshot('styled');
// Or as JSON string
const json = term.snapshotJSON('full');
```

### 5. Register Event Handlers

All terminal events are exposed as JavaScript callbacks:

```javascript
// Bell event (BEL character received)
term.on('bell', () => {
    console.log('Bell!');
    // Play a sound, flash the screen, etc.
});

// Title change (OSC 0/1/2)
term.on('title', ({ event, title }) => {
    // event: "set", "push", or "pop"
    document.title = title;
});

// Clipboard operations (OSC 52)
term.on('clipboard', ({ event, clipboard, data }) => {
    if (event === 'write') {
        navigator.clipboard.writeText(data);
    }
    // For 'read' events, return data from your handler
});

// Desktop notifications (OSC 99 - Kitty protocol)
term.on('notification', (payload) => {
    new Notification(payload.title, { body: payload.body });
});

// PTY responses (DSR, DA, etc.)
term.on('ptyWrite', (data) => {
    // data is Uint8Array - send back to your PTY
    websocket.send(data);
});

// Size queries (for pixel-level positioning)
term.on('size', (query) => {
    if (query === 'window') {
        return { width: window.innerWidth, height: window.innerHeight };
    } else { // 'cell'
        return { width: 10, height: 20 };
    }
});

// Semantic prompt marks (OSC 133 - shell integration)
term.on('semanticPrompt', ({ mark, exitCode }) => {
    // mark: 0=PromptStart, 1=PromptEnd, 2=CommandStart, 3=CommandEnd
    console.log('Prompt mark:', Terminal.PromptMark[mark], 'exit:', exitCode);
});

// APC sequences (Application Program Command)
term.on('apc', (data) => {
    // data is Uint8Array
    console.log('APC:', new TextDecoder().decode(data));
});

// PM sequences (Privacy Message)
term.on('pm', (data) => {
    // data is Uint8Array
    console.log('PM:', new TextDecoder().decode(data));
});

// SOS sequences (Start of String)
term.on('sos', (data) => {
    // data is Uint8Array
    console.log('SOS:', new TextDecoder().decode(data));
});

// Scrollback events (for custom scrollback storage)
term.on('scrollback', (event, data) => {
    switch (event) {
        case 'push':
            // data is array of cells - line scrolled off top
            storage.push(data);
            break;
        case 'pop':
            // Should return most recent line or null
            return storage.pop();
        case 'len':
            // Should return current count
            return storage.length;
        case 'line':
            // data is index, should return line or null
            return storage[data];
        case 'clear':
            storage = [];
            break;
        case 'setMaxLines':
            maxLines = data;
            break;
        case 'maxLines':
            return maxLines;
    }
});

// Recording events (for capturing raw terminal input)
term.on('recording', (event, data) => {
    switch (event) {
        case 'record':
            // data is Uint8Array - raw bytes before parsing
            recorder.push(data);
            break;
        case 'data':
            // Should return all recorded bytes as Uint8Array
            return recorder.getData();
        case 'clear':
            recorder.clear();
            break;
    }
});
```

## API Reference

### Terminal Lifecycle

| Method | Description |
|--------|-------------|
| `Terminal.init(wasmPath)` | Initialize WASM module (async, call once) |
| `new Terminal(rows, cols)` | Create terminal instance |
| `term.destroy()` | Destroy terminal instance |

### Input/Output

| Method | Description |
|--------|-------------|
| `term.write(Uint8Array)` | Write raw bytes |
| `term.writeString(string)` | Write string (ANSI sequences supported) |
| `term.getString()` | Get visible content as string |
| `term.lineContent(row)` | Get single line content |
| `term.cell(row, col)` | Get cell at position |
| `term.snapshot(detail)` | Get terminal snapshot ("text", "styled", "full") |

### Dimensions

| Property/Method | Description |
|-----------------|-------------|
| `term.rows` | Number of rows |
| `term.cols` | Number of columns |
| `term.resize(rows, cols)` | Resize terminal |

### Cursor

| Property | Description |
|----------|-------------|
| `term.cursorPos` | `{row, col}` position |
| `term.cursorVisible` | Is cursor visible? |
| `term.cursorStyle` | Cursor style (see `Terminal.CursorStyle`) |

### State

| Property/Method | Description |
|-----------------|-------------|
| `term.title` | Window title |
| `term.isAlternateScreen` | Is alternate buffer active? |
| `term.hasMode(mode)` | Check terminal mode (see `Terminal.Mode`) |
| `term.scrollRegion` | `{top, bottom}` scroll region |

### Scrollback

| Property/Method | Description |
|-----------------|-------------|
| `term.scrollbackLen` | Number of scrollback lines |
| `term.scrollbackLine(index)` | Get scrollback line (0 = oldest) |
| `term.clearScrollback()` | Clear scrollback history |

### Selection

| Method | Description |
|--------|-------------|
| `term.setSelection(r1, c1, r2, c2)` | Set text selection |
| `term.clearSelection()` | Clear selection |
| `term.hasSelection` | Is selection active? |
| `term.getSelectedText()` | Get selected text |

### Dirty Tracking

| Property/Method | Description |
|-----------------|-------------|
| `term.hasDirty` | Any cells modified? |
| `term.dirtyCells()` | Get modified cell positions |
| `term.clearDirty()` | Reset dirty flags |

### Search

| Method | Description |
|--------|-------------|
| `term.search(pattern)` | Find in visible screen |
| `term.searchScrollback(pattern)` | Find in scrollback |

### Shell Integration (OSC 133)

| Property/Method | Description |
|-----------------|-------------|
| `term.promptMarks` | Get all prompt marks |
| `term.getLastCommandOutput()` | Get last command output |
| `term.workingDirectory` | Current directory (OSC 7) |
| `term.workingDirectoryPath` | Directory path only |

### User Variables (OSC 1337)

| Method | Description |
|--------|-------------|
| `term.getUserVar(name)` | Get variable value |
| `term.getUserVars()` | Get all variables |

### Images

| Property | Description |
|----------|-------------|
| `term.imageCount` | Number of stored images |
| `term.imagePlacementCount` | Number of placements |
| `term.imageUsedMemory` | Memory usage in bytes |
| `term.sixelEnabled` | Sixel protocol enabled? |
| `term.kittyEnabled` | Kitty protocol enabled? |

## Cell Object

```javascript
{
    char: "A",           // Character (string)
    fg: {r, g, b, a},    // Foreground color
    bg: {r, g, b, a},    // Background color
    bold: false,
    dim: false,
    italic: false,
    underline: false,
    blink: false,
    reverse: false,
    hidden: false,
    strike: false,
    wideChar: false,
    hyperlink: {         // Optional
        id: "link-1",
        uri: "https://..."
    }
}
```

## Constants

### Terminal.Mode

```javascript
Terminal.Mode.CursorKeys
Terminal.Mode.Insert
Terminal.Mode.Origin
Terminal.Mode.LineWrap
Terminal.Mode.ShowCursor
Terminal.Mode.BracketedPaste
// ... and more
```

### Terminal.CursorStyle

```javascript
Terminal.CursorStyle.BlinkingBlock    // 0
Terminal.CursorStyle.SteadyBlock      // 1
Terminal.CursorStyle.BlinkingUnderline // 2
Terminal.CursorStyle.SteadyUnderline  // 3
Terminal.CursorStyle.BlinkingBar      // 4
Terminal.CursorStyle.SteadyBar        // 5
```

### Terminal.PromptMark

```javascript
Terminal.PromptMark.PromptStart   // 0
Terminal.PromptMark.PromptEnd     // 1
Terminal.PromptMark.CommandStart  // 2
Terminal.PromptMark.CommandEnd    // 3
```

## Example: Simple Terminal Renderer

```javascript
async function main() {
    await Terminal.init();
    const term = new Terminal(24, 80);

    // Handle bell
    term.on('bell', () => {
        document.body.style.background = 'red';
        setTimeout(() => document.body.style.background = '', 100);
    });

    // Handle title
    term.on('title', ({ title }) => {
        document.title = title;
    });

    // Write some content
    term.writeString('\x1b[1;32mHello World!\x1b[0m\r\n');

    // Render to canvas/DOM
    function render() {
        const container = document.getElementById('terminal');
        let html = '';

        for (let row = 0; row < term.rows; row++) {
            for (let col = 0; col < term.cols; col++) {
                const cell = term.cell(row, col);
                const char = cell.char || ' ';
                const fg = cell.fg;

                html += `<span style="color:rgb(${fg.r},${fg.g},${fg.b})">${char}</span>`;
            }
            html += '\n';
        }

        container.innerHTML = html;
        term.clearDirty();
    }

    render();
}

main();
```

## Connecting to a WebSocket PTY

```javascript
const ws = new WebSocket('ws://localhost:8080/pty');
const term = new Terminal(24, 80);

// Terminal -> PTY
term.on('ptyWrite', (data) => {
    ws.send(data);
});

// PTY -> Terminal
ws.onmessage = (event) => {
    const data = new Uint8Array(event.data);
    term.write(data);
    render();
};

// Handle resize
function resize(rows, cols) {
    term.resize(rows, cols);
    ws.send(JSON.stringify({ type: 'resize', rows, cols }));
}
```

## License

MIT
