package headlessterm

// Buffer stores a 2D grid of cells and tracks line wrapping state.
// Supports optional scrollback storage for lines scrolled off the top.
type Buffer struct {
	rows       int
	cols       int
	cells      [][]Cell
	wrapped    []bool // tracks if each line was wrapped (vs explicit newline)
	tabStop    []bool
	scrollback ScrollbackProvider
	hasDirty   bool
}

// NewBuffer creates a buffer with the given dimensions and no scrollback.
func NewBuffer(rows, cols int) *Buffer {
	return NewBufferWithStorage(rows, cols, NoopScrollback{})
}

// NewBufferWithStorage creates a buffer with custom scrollback storage.
// Tab stops are initialized every 8 columns.
func NewBufferWithStorage(rows, cols int, storage ScrollbackProvider) *Buffer {
	b := &Buffer{
		rows:       rows,
		cols:       cols,
		cells:      make([][]Cell, rows),
		wrapped:    make([]bool, rows),
		tabStop:    make([]bool, cols),
		scrollback: storage,
	}

	for i := range b.cells {
		b.cells[i] = make([]Cell, cols)
		for j := range b.cells[i] {
			b.cells[i][j] = NewCell()
		}
	}

	// Set default tab stops every 8 columns
	for i := 0; i < cols; i += 8 {
		b.tabStop[i] = true
	}

	return b
}

// Rows returns the buffer height in character rows.
func (b *Buffer) Rows() int {
	return b.rows
}

// Cols returns the buffer width in character columns.
func (b *Buffer) Cols() int {
	return b.cols
}

// Cell returns a pointer to the cell at (row, col).
// Returns nil if coordinates are out of bounds.
func (b *Buffer) Cell(row, col int) *Cell {
	if row < 0 || row >= b.rows || col < 0 || col >= b.cols {
		return nil
	}
	return &b.cells[row][col]
}

// SetCell replaces the cell at (row, col) and marks it dirty.
// Does nothing if coordinates are out of bounds.
func (b *Buffer) SetCell(row, col int, cell Cell) {
	if row < 0 || row >= b.rows || col < 0 || col >= b.cols {
		return
	}
	cell.MarkDirty()
	b.cells[row][col] = cell
	b.hasDirty = true
}

// MarkDirty marks the cell at (row, col) as modified.
// Does nothing if coordinates are out of bounds.
func (b *Buffer) MarkDirty(row, col int) {
	if row < 0 || row >= b.rows || col < 0 || col >= b.cols {
		return
	}
	b.cells[row][col].MarkDirty()
	b.hasDirty = true
}

// HasDirty returns true if any cell has been modified since the last ClearAllDirty call.
func (b *Buffer) HasDirty() bool {
	return b.hasDirty
}

// DirtyCells returns positions of all modified cells.
func (b *Buffer) DirtyCells() []Position {
	var positions []Position
	for row := range b.cells {
		for col := range b.cells[row] {
			if b.cells[row][col].IsDirty() {
				positions = append(positions, Position{Row: row, Col: col})
			}
		}
	}
	return positions
}

// ClearAllDirty resets the dirty state of all cells.
func (b *Buffer) ClearAllDirty() {
	for row := range b.cells {
		for col := range b.cells[row] {
			b.cells[row][col].ClearDirty()
		}
	}
	b.hasDirty = false
}

// ClearRow resets all cells in the row to default state and marks them dirty.
func (b *Buffer) ClearRow(row int) {
	if row < 0 || row >= b.rows {
		return
	}
	for col := range b.cells[row] {
		b.cells[row][col].Reset()
		b.cells[row][col].MarkDirty()
	}
	b.hasDirty = true
}

// ClearRowRange resets cells in the row from startCol (inclusive) to endCol (exclusive).
func (b *Buffer) ClearRowRange(row, startCol, endCol int) {
	if row < 0 || row >= b.rows {
		return
	}
	if startCol < 0 {
		startCol = 0
	}
	if endCol > b.cols {
		endCol = b.cols
	}
	for col := startCol; col < endCol; col++ {
		b.cells[row][col].Reset()
		b.cells[row][col].MarkDirty()
	}
	b.hasDirty = true
}

// ClearAll resets all cells in the buffer to default state.
func (b *Buffer) ClearAll() {
	for row := range b.cells {
		b.ClearRow(row)
	}
}

// ScrollUp shifts lines up by n positions within [top, bottom).
// Lines scrolled off the top are pushed to scrollback if enabled and top==0.
// Bottom lines are cleared and marked dirty.
func (b *Buffer) ScrollUp(top, bottom, n int) {
	if n <= 0 || top >= bottom {
		return
	}
	if top < 0 {
		top = 0
	}
	if bottom > b.rows {
		bottom = b.rows
	}

	if n > bottom-top {
		n = bottom - top
	}

	// Save lines to scrollback if enabled and scrolling from top
	if b.scrollback != nil && b.scrollback.MaxLines() > 0 && top == 0 {
		for i := 0; i < n; i++ {
			b.scrollback.Push(b.cells[i])
		}
	}

	// Move lines up (including wrapped flags)
	for row := top; row < bottom-n; row++ {
		b.cells[row] = b.cells[row+n]
		b.wrapped[row] = b.wrapped[row+n]
		for col := range b.cells[row] {
			b.cells[row][col].MarkDirty()
		}
	}

	// Clear the bottom lines
	for row := bottom - n; row < bottom; row++ {
		b.cells[row] = make([]Cell, b.cols)
		b.wrapped[row] = false
		for col := range b.cells[row] {
			b.cells[row][col] = NewCell()
			b.cells[row][col].MarkDirty()
		}
	}
	b.hasDirty = true
}

// ScrollDown shifts lines down by n positions within [top, bottom).
// Top lines are cleared and marked dirty.
func (b *Buffer) ScrollDown(top, bottom, n int) {
	if n <= 0 || top >= bottom {
		return
	}
	if top < 0 {
		top = 0
	}
	if bottom > b.rows {
		bottom = b.rows
	}

	if n > bottom-top {
		n = bottom - top
	}

	// Move lines down (including wrapped flags)
	for row := bottom - 1; row >= top+n; row-- {
		b.cells[row] = b.cells[row-n]
		b.wrapped[row] = b.wrapped[row-n]
		for col := 0; col < b.cols; col++ {
			b.cells[row][col].MarkDirty()
		}
	}

	// Clear the top lines
	for row := top; row < top+n; row++ {
		b.cells[row] = make([]Cell, b.cols)
		b.wrapped[row] = false
		for col := 0; col < b.cols; col++ {
			b.cells[row][col] = NewCell()
			b.cells[row][col].MarkDirty()
		}
	}
	b.hasDirty = true
}

// InsertLines inserts n blank lines at row, shifting existing lines down.
// Equivalent to ScrollDown(row, bottom, n).
func (b *Buffer) InsertLines(row, n, bottom int) {
	if row < 0 || row >= bottom || n <= 0 {
		return
	}
	b.ScrollDown(row, bottom, n)
}

// DeleteLines removes n lines at row, shifting remaining lines up.
// Equivalent to ScrollUp(row, bottom, n).
func (b *Buffer) DeleteLines(row, n, bottom int) {
	if row < 0 || row >= bottom || n <= 0 {
		return
	}
	b.ScrollUp(row, bottom, n)
}

// InsertBlanks inserts n blank cells at (row, col), shifting existing characters right.
func (b *Buffer) InsertBlanks(row, col, n int) {
	if row < 0 || row >= b.rows || col < 0 || col >= b.cols || n <= 0 {
		return
	}

	// Shift characters to the right
	for c := b.cols - 1; c >= col+n; c-- {
		b.cells[row][c] = b.cells[row][c-n]
		b.cells[row][c].MarkDirty()
	}

	// Clear the inserted positions
	for c := col; c < col+n && c < b.cols; c++ {
		b.cells[row][c].Reset()
		b.cells[row][c].MarkDirty()
	}
	b.hasDirty = true
}

// DeleteChars removes n characters at (row, col), shifting remaining characters left.
func (b *Buffer) DeleteChars(row, col, n int) {
	if row < 0 || row >= b.rows || col < 0 || col >= b.cols || n <= 0 {
		return
	}

	// Shift characters to the left
	for c := col; c < b.cols-n; c++ {
		b.cells[row][c] = b.cells[row][c+n]
		b.cells[row][c].MarkDirty()
	}

	// Clear the end of the line
	for c := b.cols - n; c < b.cols; c++ {
		if c >= 0 {
			b.cells[row][c].Reset()
			b.cells[row][c].MarkDirty()
		}
	}
	b.hasDirty = true
}

// Resize changes buffer dimensions, preserving existing cells where possible.
// Content is kept at the top-left corner. When shrinking, bottom/right content is lost.
// When growing, new empty cells are added at the bottom/right.
// Tab stops are extended if columns increase.
func (b *Buffer) Resize(rows, cols int) {
	if rows <= 0 || cols <= 0 {
		return
	}

	newCells := make([][]Cell, rows)
	for i := range newCells {
		newCells[i] = make([]Cell, cols)
		for j := range newCells[i] {
			if i < b.rows && j < b.cols {
				newCells[i][j] = b.cells[i][j]
			} else {
				newCells[i][j] = NewCell()
			}
			newCells[i][j].MarkDirty()
		}
	}

	// Resize wrapped tracking
	newWrapped := make([]bool, rows)
	copy(newWrapped, b.wrapped)

	b.cells = newCells
	b.wrapped = newWrapped
	b.rows = rows
	b.cols = cols
	b.hasDirty = true

	// Resize tab stops
	newTabStop := make([]bool, cols)
	copy(newTabStop, b.tabStop)
	for i := len(b.tabStop); i < cols; i += 8 {
		newTabStop[i] = true
	}
	b.tabStop = newTabStop
}

// SetTabStop enables a tab stop at the specified column.
func (b *Buffer) SetTabStop(col int) {
	if col >= 0 && col < b.cols {
		b.tabStop[col] = true
	}
}

// ClearTabStop disables the tab stop at the specified column.
func (b *Buffer) ClearTabStop(col int) {
	if col >= 0 && col < b.cols {
		b.tabStop[col] = false
	}
}

// ClearAllTabStops disables all tab stops.
func (b *Buffer) ClearAllTabStops() {
	for i := range b.tabStop {
		b.tabStop[i] = false
	}
}

// NextTabStop returns the column index of the next enabled tab stop after col.
// Returns the last column if no tab stop is found.
func (b *Buffer) NextTabStop(col int) int {
	for c := col + 1; c < b.cols; c++ {
		if b.tabStop[c] {
			return c
		}
	}
	return b.cols - 1
}

// PrevTabStop returns the column index of the previous enabled tab stop before col.
// Returns 0 if no tab stop is found.
func (b *Buffer) PrevTabStop(col int) int {
	for c := col - 1; c >= 0; c-- {
		if b.tabStop[c] {
			return c
		}
	}
	return 0
}

// FillWithE fills all cells with 'E' (used by DECALN alignment test pattern).
func (b *Buffer) FillWithE() {
	for row := range b.cells {
		for col := range b.cells[row] {
			b.cells[row][col].Reset()
			b.cells[row][col].Char = 'E'
			b.cells[row][col].MarkDirty()
		}
	}
	b.hasDirty = true
}

// ScrollbackLen returns the number of lines stored in scrollback.
func (b *Buffer) ScrollbackLen() int {
	if b.scrollback == nil {
		return 0
	}
	return b.scrollback.Len()
}

// ScrollbackLine returns a line from scrollback, where 0 is the oldest line.
// Returns nil if index is out of range or scrollback is disabled.
func (b *Buffer) ScrollbackLine(index int) []Cell {
	if b.scrollback == nil {
		return nil
	}
	return b.scrollback.Line(index)
}

// ClearScrollback removes all stored scrollback lines.
func (b *Buffer) ClearScrollback() {
	if b.scrollback != nil {
		b.scrollback.Clear()
	}
}

// SetMaxScrollback sets the maximum number of scrollback lines to retain.
func (b *Buffer) SetMaxScrollback(max int) {
	if b.scrollback != nil {
		b.scrollback.SetMaxLines(max)
	}
}

// MaxScrollback returns the current maximum scrollback capacity.
func (b *Buffer) MaxScrollback() int {
	if b.scrollback == nil {
		return 0
	}
	return b.scrollback.MaxLines()
}

// SetScrollbackProvider replaces the scrollback storage implementation.
func (b *Buffer) SetScrollbackProvider(storage ScrollbackProvider) {
	b.scrollback = storage
}

// ScrollbackProvider returns the current scrollback storage implementation.
func (b *Buffer) ScrollbackProvider() ScrollbackProvider {
	return b.scrollback
}

// LineContent returns the text content of a line, trimming trailing spaces.
// Wide character spacers are skipped. Returns empty string if the line is empty or out of bounds.
func (b *Buffer) LineContent(row int) string {
	if row < 0 || row >= b.rows {
		return ""
	}

	// Find the last non-space character
	lastNonSpace := -1
	for col := b.cols - 1; col >= 0; col-- {
		cell := &b.cells[row][col]
		if cell.Char != ' ' && cell.Char != 0 && !cell.IsWideSpacer() {
			lastNonSpace = col
			break
		}
	}

	if lastNonSpace < 0 {
		return ""
	}

	runes := make([]rune, 0, lastNonSpace+1)
	for col := range b.cells[row][:lastNonSpace+1] {
		cell := &b.cells[row][col]
		if cell.IsWideSpacer() {
			continue
		}
		if cell.Char == 0 {
			runes = append(runes, ' ')
		} else {
			runes = append(runes, cell.Char)
		}
	}

	return string(runes)
}

// --- Auto Resize ---

// GrowRows appends n new rows to the bottom of the buffer.
// New cells are initialized to default state and marked dirty.
func (b *Buffer) GrowRows(n int) {
	if n <= 0 {
		return
	}

	newRows := b.rows + n
	newCells := make([][]Cell, newRows)
	newWrapped := make([]bool, newRows)

	// Copy existing rows
	copy(newCells, b.cells)
	copy(newWrapped, b.wrapped)

	// Initialize new rows
	for i := b.rows; i < newRows; i++ {
		newCells[i] = make([]Cell, b.cols)
		for j := range newCells[i] {
			newCells[i][j] = NewCell()
			newCells[i][j].MarkDirty()
		}
	}

	b.cells = newCells
	b.wrapped = newWrapped
	b.rows = newRows
	b.hasDirty = true
}

// GrowCols expands a single row to at least minCols columns.
// Does nothing if the row is already wider. Tab stops are extended if needed.
func (b *Buffer) GrowCols(row, minCols int) {
	if row < 0 || row >= b.rows {
		return
	}
	if minCols <= len(b.cells[row]) {
		return
	}

	// Expand just this row
	newCells := make([]Cell, minCols)
	copy(newCells, b.cells[row])
	for j := len(b.cells[row]); j < minCols; j++ {
		newCells[j] = NewCell()
		newCells[j].MarkDirty()
	}
	b.cells[row] = newCells

	// Track max cols for reference
	if minCols > b.cols {
		b.cols = minCols
		// Expand tabstops
		newTabStop := make([]bool, minCols)
		copy(newTabStop, b.tabStop)
		for i := len(b.tabStop); i < minCols; i += 8 {
			newTabStop[i] = true
		}
		b.tabStop = newTabStop
	}

	b.hasDirty = true
}

// --- Wrapped Line Tracking ---

// IsWrapped returns true if the line was wrapped due to column overflow.
func (b *Buffer) IsWrapped(row int) bool {
	if row < 0 || row >= b.rows {
		return false
	}
	return b.wrapped[row]
}

// SetWrapped sets whether the line was wrapped or ended with an explicit newline.
func (b *Buffer) SetWrapped(row int, wrapped bool) {
	if row < 0 || row >= b.rows {
		return
	}
	b.wrapped[row] = wrapped
}

// Position identifies a cell location in the terminal grid (0-based).
type Position struct {
	Row int
	Col int
}

// Before returns true if this position comes before other in reading order (top-to-bottom, left-to-right).
func (p Position) Before(other Position) bool {
	if p.Row < other.Row {
		return true
	}
	if p.Row == other.Row && p.Col < other.Col {
		return true
	}
	return false
}

// Equal returns true if both row and column match.
func (p Position) Equal(other Position) bool {
	return p.Row == other.Row && p.Col == other.Col
}
