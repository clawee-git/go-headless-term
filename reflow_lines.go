package headlessterm

import (
	"image/color"
	"slices"
)

type reflowRow struct {
	cells   []Cell
	wrapped bool
}

type reflowLine struct {
	cells []Cell
}

type reflowPoint struct {
	row, col       *int
	oldRow, oldCol int
	srcRow         int
	line, offset   int
	placed         bool
	newRow, newCol int
}

func newReflowPoint(row, col *int, scrollbackLen int) *reflowPoint {
	return &reflowPoint{row: row, col: col, oldRow: *row, oldCol: *col, srcRow: scrollbackLen + *row, line: -1}
}

func isDefaultColor(c color.Color, name int) bool {
	n, ok := c.(*NamedColor)
	return ok && n.Name == name
}

func isBlankCell(c Cell) bool {
	return (c.Char == ' ' || c.Char == 0) &&
		c.Flags&^CellFlagDirty == 0 &&
		c.Hyperlink == nil &&
		c.Image == nil &&
		c.UnderlineColor == nil &&
		isDefaultColor(c.Fg, NamedColorForeground) &&
		isDefaultColor(c.Bg, NamedColorBackground)
}

func reflowSourceRows(b *Buffer, wp WrapScrollbackProvider) []reflowRow {
	var rows []reflowRow
	if wp != nil {
		for i := range wp.Len() {
			rows = append(rows, reflowRow{cells: slices.Clone(wp.Line(i)), wrapped: wp.LineWrapped(i)})
		}
	}
	for r := range b.rows {
		rows = append(rows, reflowRow{cells: slices.Clone(b.cells[r]), wrapped: b.wrapped[r]})
	}
	return rows
}

func dropWidePad(cur, next []Cell) []Cell {
	if len(cur) > 0 && len(next) > 0 && next[0].HasFlag(CellFlagWideChar) && isBlankCell(cur[len(cur)-1]) {
		return cur[:len(cur)-1]
	}
	return cur
}

func joinRows(rows []reflowRow, points []*reflowPoint) []reflowLine {
	var lines []reflowLine
	var cur []Cell
	for i, row := range rows {
		if i > 0 && rows[i-1].wrapped {
			cur = dropWidePad(cur, row.cells)
		} else if i > 0 {
			lines = append(lines, reflowLine{cells: cur})
			cur = nil
		}
		for _, p := range points {
			if p.srcRow == i {
				p.line, p.offset = len(lines), len(cur)+p.oldCol
			}
		}
		cur = append(cur, row.cells...)
	}
	if len(rows) > 0 {
		lines = append(lines, reflowLine{cells: cur})
	}
	return lines
}

func fitLines(lines []reflowLine, points []*reflowPoint) {
	minLen := make([]int, len(lines))
	for _, p := range points {
		if p.line >= 0 {
			minLen[p.line] = max(minLen[p.line], p.offset)
		}
	}
	for i := range lines {
		n := len(lines[i].cells)
		for n > minLen[i] && n > 0 && isBlankCell(lines[i].cells[n-1]) {
			n--
		}
		lines[i].cells = lines[i].cells[:n]
		for len(lines[i].cells) < minLen[i] {
			lines[i].cells = append(lines[i].cells, NewCell())
		}
	}
}

func lastKeptLine(lines []reflowLine, cursor *reflowPoint) int {
	last := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i].cells) > 0 {
			last = i
			break
		}
	}
	if cursor != nil && cursor.line > last {
		last = cursor.line
	}
	return last
}
