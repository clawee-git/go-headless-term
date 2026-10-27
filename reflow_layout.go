package headlessterm

import "slices"

type cellPos struct {
	row, col int
}

type placedRows struct {
	cells   [][]Cell
	wrapped []bool
	cols    int
}

func (p *placedRows) openRow() {
	row := make([]Cell, p.cols)
	for i := range row {
		row[i] = NewCell()
	}
	p.cells = append(p.cells, row)
	p.wrapped = append(p.wrapped, false)
}

func unitAt(cells []Cell, i, cols int) (count, width int) {
	if cells[i].HasFlag(CellFlagWideChar) && i+1 < len(cells) && cells[i+1].HasFlag(CellFlagWideCharSpacer) {
		return 2, min(2, cols)
	}
	return 1, 1
}

func (p *placedRows) put(unit []Cell, width int, at cellPos, pos []cellPos) {
	for k, c := range unit {
		pos[k] = cellPos{at.row, min(at.col+k, p.cols-1)}
		if width < len(unit) {
			if k > 0 {
				continue
			}
			c.Flags &^= CellFlagWideChar
		}
		p.cells[at.row][at.col+k] = c
	}
}

func (p *placedRows) layout(cells []Cell, pos []cellPos) {
	p.openRow()
	col := 0
	for i := 0; i < len(cells); {
		count, width := unitAt(cells, i, p.cols)
		if col+width > p.cols {
			p.wrapped[len(p.wrapped)-1] = true
			p.openRow()
			col = 0
		}
		p.put(cells[i:i+count], width, cellPos{len(p.cells) - 1, col}, pos[i:i+count])
		col += width
		i += count
	}
	pos[len(cells)] = cellPos{len(p.cells) - 1, col}
}

func layoutLines(placed *placedRows, lines []reflowLine, points []*reflowPoint) {
	var pos []cellPos
	for li, line := range lines {
		pos = slices.Grow(pos[:0], len(line.cells)+1)[:len(line.cells)+1]
		placed.layout(line.cells, pos)
		for _, p := range points {
			if p.line == li {
				p.newRow, p.newCol, p.placed = pos[p.offset].row, pos[p.offset].col, true
			}
		}
	}
}
