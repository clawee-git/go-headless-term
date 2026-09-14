package headlessterm

import "slices"

// primaryCursorRow returns the primary screen's cursor row: the live cursor
// when the primary is active, the row ?1049h saved while the alternate screen
// is active, or nil when there is no such row. Caller must hold the lock.
func (t *Terminal) primaryCursorRow() *int {
	if t.activeBuffer == t.primaryBuffer {
		return &t.cursor.Row
	}
	if t.savedCursor != nil {
		return &t.savedCursor.Row
	}
	return nil
}

// shrinkPrimaryRows scrolls the primary screen into scrollback before a row
// shrink when its cursor would fall below the new last row. Caller must hold
// the lock and resize the buffers afterwards.
func (t *Terminal) shrinkPrimaryRows(oldRows, rows int) {
	lines := oldRows - rows
	cursorRow := t.primaryCursorRow()
	if cursorRow == nil || *cursorRow < rows {
		if t.activeBuffer != t.primaryBuffer {
			t.primaryRowsCutOnAlternate += lines
		}
		return
	}
	t.primaryBuffer.ScrollUp(0, oldRows, lines)
	*cursorRow = max(*cursorRow-lines, 0)
}

// growPrimaryRows pulls scrollback lines back onto the top of the primary
// screen after a row grow, first handing back rows an alternate-screen shrink
// cut without scrolling. Caller must hold the lock and have resized the buffers.
func (t *Terminal) growPrimaryRows(oldRows, rows int) {
	growth := rows - oldRows
	restored := min(growth, t.primaryRowsCutOnAlternate)
	t.primaryRowsCutOnAlternate -= restored

	lines := t.popScrollbackLines(growth - restored)
	if len(lines) == 0 {
		return
	}
	t.primaryBuffer.ScrollDown(0, rows, len(lines))
	for i, line := range lines {
		for col, cell := range line {
			if col < t.cols {
				t.primaryBuffer.SetCell(i, col, cell)
			}
		}
	}
	if cursorRow := t.primaryCursorRow(); cursorRow != nil {
		*cursorRow += len(lines)
	}
}

// popScrollbackLines pops up to n lines from the primary scrollback and returns
// them oldest first.
func (t *Terminal) popScrollbackLines(n int) [][]Cell {
	scrollback := t.primaryBuffer.ScrollbackProvider()
	if scrollback == nil || n <= 0 {
		return nil
	}
	n = min(n, scrollback.Len())
	lines := make([][]Cell, 0, n)
	for range n {
		line := scrollback.Pop()
		if line == nil {
			break
		}
		lines = append(lines, line)
	}
	slices.Reverse(lines)
	return lines
}

// scrollActiveRegionUp scrolls the active buffer's scroll region up by n lines
// for output that runs past the bottom margin (line feeds, image placement).
// When that region is the whole primary screen, the scroll happened only
// because the screen is shorter: the lines would have fit in the rows an
// alternate-screen shrink cut, so they use up primaryRowsCutOnAlternate and a
// later grow pops them back instead of handing back blank rows. Scrolls a
// terminal performs the same at any height (SU, a region ending above the last
// row) and Resize's own shrink scroll do not use up the counter.
// Caller must hold the lock.
func (t *Terminal) scrollActiveRegionUp(n int) {
	t.activeBuffer.ScrollUp(t.scrollTop, t.scrollBottom, n)
	if t.activeBuffer != t.primaryBuffer || t.scrollTop != 0 || t.scrollBottom != t.rows || n <= 0 {
		return
	}
	scrolled := min(n, t.scrollBottom-t.scrollTop)
	t.primaryRowsCutOnAlternate = max(0, t.primaryRowsCutOnAlternate-scrolled)
}
