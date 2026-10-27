package headlessterm

func (t *Terminal) SetReflow(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reflow = enabled
}

func (t *Terminal) reflowPoints(scrollbackLen int) (cursor *reflowPoint, all []*reflowPoint) {
	if t.activeBuffer == t.primaryBuffer {
		cursor = newReflowPoint(&t.cursor.Row, &t.cursor.Col, scrollbackLen)
		all = append(all, cursor)
		if t.savedCursor != nil {
			all = append(all, newReflowPoint(&t.savedCursor.Row, &t.savedCursor.Col, scrollbackLen))
		}
		return cursor, all
	}
	if t.savedCursor != nil {
		cursor = newReflowPoint(&t.savedCursor.Row, &t.savedCursor.Col, scrollbackLen)
		all = append(all, cursor)
	}
	return cursor, all
}

func (t *Terminal) reflowPrimary(rows, cols int) {
	b := t.primaryBuffer
	wp, _ := b.scrollback.(WrapScrollbackProvider)
	src := reflowSourceRows(b, wp)
	cursor, points := t.reflowPoints(len(src) - b.rows)
	lines := joinRows(src, points)
	fitLines(lines, points)
	placed := &placedRows{cols: cols}
	layoutLines(placed, lines[:lastKeptLine(lines, cursor)+1], points)
	start := max(0, len(placed.cells)-rows)
	b.Resize(rows, cols)
	b.installRows(placed, start)
	if wp != nil {
		replaceScrollback(wp, placed, start)
	} else {
		pushAbove(b.scrollback, placed, start)
	}
	for _, p := range points {
		p.settle(start, rows, cols)
	}
	t.primaryRowsCutOnAlternate = 0
}

func (p *reflowPoint) settle(start, rows, cols int) {
	row, col := p.newRow-start, p.newCol
	if !p.placed {
		row, col = min(p.oldRow, rows-1), min(p.oldCol, cols-1)
	}
	if row < 0 {
		row, col = 0, 0
	}
	*p.row, *p.col = row, min(col, cols)
}

func (b *Buffer) installRows(placed *placedRows, start int) {
	for r := range b.rows {
		if start+r >= len(placed.cells) {
			placed.openRow()
		}
		b.cells[r] = placed.cells[start+r]
		b.wrapped[r] = placed.wrapped[start+r]
	}
	for r := range b.cells {
		for c := range b.cells[r] {
			b.cells[r][c].MarkDirty()
		}
	}
	b.hasDirty = true
}

func replaceScrollback(wp WrapScrollbackProvider, placed *placedRows, start int) {
	above, flags := placed.cells[:start], placed.wrapped[:start]
	limit := wp.MaxLines()
	if limit <= 0 {
		above, flags = nil, nil
	} else if len(above) > limit {
		above, flags = above[len(above)-limit:], flags[len(flags)-limit:]
	}
	wp.ReplaceWrapped(above, flags)
}

func pushAbove(sb ScrollbackProvider, placed *placedRows, start int) {
	if sb == nil {
		return
	}
	for _, row := range placed.cells[:start] {
		sb.Push(row)
	}
}

func (t *Terminal) markWidePad(width int) {
	if !t.reflow || width != 2 || t.cursor.Col >= t.cols {
		return
	}
	if cell := t.activeBuffer.Cell(t.cursor.Row, t.cursor.Col); cell != nil {
		cell.wrapPad = true
	}
}
