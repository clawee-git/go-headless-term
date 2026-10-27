package headlessterm

import (
	"reflect"
	"strings"
	"testing"
)

type wrapCounting struct {
	*MemoryScrollback
	pushes, pops, pushWrapped, popWrapped, replaces int
}

func newWrapCounting(max int) *wrapCounting {
	return &wrapCounting{MemoryScrollback: NewMemoryScrollback(max)}
}

func (w *wrapCounting) Push(line []Cell) {
	w.pushes++
	w.MemoryScrollback.Push(line)
}

func (w *wrapCounting) Pop() []Cell {
	w.pops++
	return w.MemoryScrollback.Pop()
}

func (w *wrapCounting) PushWrapped(line []Cell, wrapped bool) {
	w.pushWrapped++
	w.MemoryScrollback.PushWrapped(line, wrapped)
}

func (w *wrapCounting) PopWrapped() ([]Cell, bool) {
	w.popWrapped++
	return w.MemoryScrollback.PopWrapped()
}

func (w *wrapCounting) ReplaceWrapped(lines [][]Cell, wrapped []bool) {
	w.replaces++
	w.MemoryScrollback.ReplaceWrapped(lines, wrapped)
}

type reflowState struct {
	screen      [][]Cell
	screenWrap  []bool
	history     [][]Cell
	historyWrap []bool
	cursorRow   int
	cursorCol   int
}

func maskedCells(cells []Cell) []Cell {
	out := make([]Cell, len(cells))
	copy(out, cells)
	for i := range out {
		out[i].Flags &^= CellFlagDirty
	}
	return out
}

func captureReflow(term *Terminal, sb ScrollbackProvider) reflowState {
	var s reflowState
	for row := range term.Rows() {
		cells := make([]Cell, term.Cols())
		for col := range cells {
			cells[col] = *term.Cell(row, col)
		}
		s.screen = append(s.screen, maskedCells(cells))
		s.screenWrap = append(s.screenWrap, term.IsWrapped(row))
	}
	wp, _ := sb.(WrapScrollbackProvider)
	for i := range sb.Len() {
		s.history = append(s.history, maskedCells(sb.Line(i)))
		s.historyWrap = append(s.historyWrap, wp != nil && wp.LineWrapped(i))
	}
	s.cursorRow, s.cursorCol = term.CursorPos()
	return s
}

func requireSameReflowState(t *testing.T, got, want reflowState) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("state differs\n got screen %q wrap %v history %q wrap %v cursor %d,%d\nwant screen %q wrap %v history %q wrap %v cursor %d,%d",
			cellsText(got.screen), got.screenWrap, cellsText(got.history), got.historyWrap, got.cursorRow, got.cursorCol,
			cellsText(want.screen), want.screenWrap, cellsText(want.history), want.historyWrap, want.cursorRow, want.cursorCol)
	}
}

func cellsText(lines [][]Cell) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		var b strings.Builder
		for _, c := range line {
			if !c.HasFlag(CellFlagWideCharSpacer) {
				b.WriteRune(c.Char)
			}
		}
		out[i] = strings.TrimRight(b.String(), " ")
	}
	return out
}

func newReflowTerm(rows, cols int, sb ScrollbackProvider) *Terminal {
	return New(WithSize(rows, cols), WithScrollback(sb), WithReflow(true))
}

func screenText(term *Terminal) []string {
	out := make([]string, term.Rows())
	for row := range out {
		out[row] = term.LineContent(row)
	}
	return out
}

func wrapFlags(term *Terminal) []bool {
	out := make([]bool, term.Rows())
	for row := range out {
		out[row] = term.IsWrapped(row)
	}
	return out
}

func digits(n int) string {
	var b strings.Builder
	for i := range n {
		b.WriteRune(rune('a' + i%26))
	}
	return b.String()
}
