package headlessterm

import (
	"testing"
)

func TestNewBuffer(t *testing.T) {
	b := NewBuffer(24, 80)

	if b.Rows() != 24 {
		t.Errorf("expected 24 rows, got %d", b.Rows())
	}
	if b.Cols() != 80 {
		t.Errorf("expected 80 cols, got %d", b.Cols())
	}
}

func TestBufferCell(t *testing.T) {
	b := NewBuffer(24, 80)

	cell := b.Cell(0, 0)
	if cell == nil {
		t.Fatal("expected cell at (0,0)")
	}

	cell.Char = 'A'

	retrieved := b.Cell(0, 0)
	if retrieved.Char != 'A' {
		t.Errorf("expected 'A', got '%c'", retrieved.Char)
	}
}

func TestBufferCellOutOfBounds(t *testing.T) {
	b := NewBuffer(24, 80)

	cases := []struct {
		row, col int
		desc     string
	}{
		{-1, 0, "negative row"},
		{0, -1, "negative col"},
		{24, 0, "row >= rows"},
		{0, 80, "col >= cols"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if b.Cell(c.row, c.col) != nil {
				t.Errorf("expected nil for %s", c.desc)
			}
		})
	}
}

func TestBufferClearRow(t *testing.T) {
	b := NewBuffer(24, 80)

	b.Cell(0, 0).Char = 'A'
	b.Cell(0, 1).Char = 'B'

	b.ClearRow(0)

	if b.Cell(0, 0).Char != ' ' {
		t.Error("expected cell to be cleared")
	}
	if b.Cell(0, 1).Char != ' ' {
		t.Error("expected cell to be cleared")
	}
}

func TestBufferScroll(t *testing.T) {
	cases := []struct {
		name      string
		direction string
		wantRow0  rune
		wantLast  rune
	}{
		{"up", "up", '1', ' '},
		{"down", "down", ' ', '3'},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := NewBuffer(5, 10)
			for row := 0; row < 5; row++ {
				b.Cell(row, 0).Char = rune('0' + row)
			}

			if c.direction == "up" {
				b.ScrollUp(0, 5, 1)
			} else {
				b.ScrollDown(0, 5, 1)
			}

			if b.Cell(0, 0).Char != c.wantRow0 {
				t.Errorf("expected '%c', got '%c'", c.wantRow0, b.Cell(0, 0).Char)
			}
			if b.Cell(4, 0).Char != c.wantLast {
				t.Errorf("expected '%c', got '%c'", c.wantLast, b.Cell(4, 0).Char)
			}
		})
	}
}

func TestBufferScrollback(t *testing.T) {
	storage := &testScrollbackBuffer{lines: make([][]Cell, 0), maxLines: 100}
	b := NewBufferWithStorage(5, 10, storage)

	for row := 0; row < 5; row++ {
		b.Cell(row, 0).Char = rune('A' + row)
	}

	b.ScrollUp(0, 5, 1)

	if b.ScrollbackLen() != 1 {
		t.Errorf("expected 1 scrollback line, got %d", b.ScrollbackLen())
	}

	line := b.ScrollbackLine(0)
	if line == nil {
		t.Fatal("expected scrollback line")
	}
	if line[0].Char != 'A' {
		t.Errorf("expected 'A' in scrollback, got '%c'", line[0].Char)
	}
}

// testScrollbackBuffer is a test implementation of ScrollbackProvider
type testScrollbackBuffer struct {
	lines    [][]Cell
	maxLines int
}

func (s *testScrollbackBuffer) Push(line []Cell) {
	lineCopy := make([]Cell, len(line))
	copy(lineCopy, line)
	s.lines = append(s.lines, lineCopy)
	if s.maxLines > 0 && len(s.lines) > s.maxLines {
		s.lines = s.lines[len(s.lines)-s.maxLines:]
	}
}

func (s *testScrollbackBuffer) Len() int              { return len(s.lines) }
func (s *testScrollbackBuffer) Line(index int) []Cell { return s.lines[index] }
func (s *testScrollbackBuffer) Clear()                { s.lines = make([][]Cell, 0) }
func (s *testScrollbackBuffer) SetMaxLines(max int)   { s.maxLines = max }
func (s *testScrollbackBuffer) MaxLines() int         { return s.maxLines }

func (s *testScrollbackBuffer) Pop() []Cell {
	if len(s.lines) == 0 {
		return nil
	}
	line := s.lines[len(s.lines)-1]
	s.lines = s.lines[:len(s.lines)-1]
	return line
}

func TestBufferLineContent(t *testing.T) {
	b := NewBuffer(24, 80)

	b.Cell(0, 0).Char = 'H'
	b.Cell(0, 1).Char = 'e'
	b.Cell(0, 2).Char = 'l'
	b.Cell(0, 3).Char = 'l'
	b.Cell(0, 4).Char = 'o'

	content := b.LineContent(0)
	if content != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", content)
	}
}

func TestBufferTabStops(t *testing.T) {
	b := NewBuffer(24, 80)

	cases := []struct {
		name     string
		from     int
		forward  bool
		expected int
	}{
		{"next from 0", 0, true, 8},
		{"next from 8", 8, true, 16},
		{"prev from 16", 16, false, 8},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got int
			if c.forward {
				got = b.NextTabStop(c.from)
			} else {
				got = b.PrevTabStop(c.from)
			}
			if got != c.expected {
				t.Errorf("expected %d, got %d", c.expected, got)
			}
		})
	}
}

func TestBufferResize(t *testing.T) {
	b := NewBuffer(10, 20)

	b.Cell(0, 0).Char = 'A'
	b.Cell(5, 10).Char = 'B'

	b.Resize(20, 40)

	if b.Rows() != 20 || b.Cols() != 40 {
		t.Errorf("expected 20x40, got %dx%d", b.Rows(), b.Cols())
	}

	if b.Cell(0, 0).Char != 'A' {
		t.Error("expected content to be preserved")
	}
	if b.Cell(5, 10).Char != 'B' {
		t.Error("expected content to be preserved")
	}
}

func TestBufferDirtyTracking(t *testing.T) {
	b := NewBuffer(24, 80)

	b.ClearAllDirty()

	if b.HasDirty() {
		t.Error("expected no dirty cells")
	}

	b.MarkDirty(0, 0)

	if !b.HasDirty() {
		t.Error("expected dirty cells")
	}

	dirty := b.DirtyCells()
	if len(dirty) != 1 {
		t.Errorf("expected 1 dirty cell, got %d", len(dirty))
	}
	if dirty[0].Row != 0 || dirty[0].Col != 0 {
		t.Error("expected dirty cell at (0,0)")
	}
}

func TestBufferLineEdits(t *testing.T) {
	cases := []struct {
		name      string
		operation string
		setup     []rune
		at        int
		count     int
		want      []rune
	}{
		{
			name:      "insert blanks",
			operation: "insert",
			setup:     []rune{'A', 'B', 'C'},
			at:        1,
			count:     2,
			want:      []rune{'A', ' ', ' ', 'B', 'C'},
		},
		{
			name:      "delete chars",
			operation: "delete",
			setup:     []rune{'A', 'B', 'C', 'D'},
			at:        1,
			count:     2,
			want:      []rune{'A', 'D', ' ', ' '},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := NewBuffer(24, 80)
			for i, ch := range c.setup {
				b.Cell(0, i).Char = ch
			}

			if c.operation == "insert" {
				b.InsertBlanks(0, c.at, c.count)
			} else {
				b.DeleteChars(0, c.at, c.count)
			}

			for i, want := range c.want {
				if got := b.Cell(0, i).Char; got != want {
					t.Errorf("cell %d: expected '%c', got '%c'", i, want, got)
				}
			}
		})
	}
}

func TestBufferWrappedLineTracking(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		b := NewBuffer(5, 10)

		if b.IsWrapped(0) {
			t.Error("expected line 0 not wrapped initially")
		}

		b.SetWrapped(0, true)
		if !b.IsWrapped(0) {
			t.Error("expected line 0 to be wrapped")
		}

		b.SetWrapped(0, false)
		if b.IsWrapped(0) {
			t.Error("expected line 0 not wrapped after clear")
		}

		b.SetWrapped(-1, true)
		b.SetWrapped(100, true)
		if b.IsWrapped(-1) {
			t.Error("expected false for out of bounds")
		}
		if b.IsWrapped(100) {
			t.Error("expected false for out of bounds")
		}
	})

	t.Run("with scroll", func(t *testing.T) {
		b := NewBuffer(5, 10)

		b.SetWrapped(0, true)
		b.SetWrapped(1, false)
		b.SetWrapped(2, true)

		b.ScrollUp(0, 5, 1)

		if b.IsWrapped(0) {
			t.Error("expected line 0 not wrapped after scroll")
		}
		if !b.IsWrapped(1) {
			t.Error("expected line 1 wrapped after scroll")
		}
		if b.IsWrapped(4) {
			t.Error("expected new line not wrapped")
		}
	})
}

var bufferGrowCases = []struct {
	name      string
	axis      string
	setup     func(*Buffer)
	grow      func(*Buffer)
	checkDim  func(*Buffer) int
	expected  int
	checkPres func(*Buffer) bool
}{
	{
		name: "rows",
		axis: "rows",
		setup: func(b *Buffer) {
			b.Cell(0, 0).Char = 'A'
			b.Cell(4, 0).Char = 'E'
		},
		grow:     func(b *Buffer) { b.GrowRows(3) },
		checkDim: func(b *Buffer) int { return b.Rows() },
		expected: 8,
		checkPres: func(b *Buffer) bool {
			return b.Cell(0, 0).Char == 'A' && b.Cell(4, 0).Char == 'E' && b.Cell(7, 0).Char == ' '
		},
	},
	{
		name: "cols",
		axis: "cols",
		setup: func(b *Buffer) {
			b.Cell(0, 0).Char = 'A'
			b.Cell(0, 9).Char = 'B'
		},
		grow:     func(b *Buffer) { b.GrowCols(0, 20) },
		checkDim: func(b *Buffer) int { return b.Cols() },
		expected: 20,
		checkPres: func(b *Buffer) bool {
			return b.Cell(0, 0).Char == 'A' && b.Cell(0, 9).Char == 'B' && b.Cell(0, 15).Char == ' '
		},
	},
}

func TestBufferGrow(t *testing.T) {
	for _, c := range bufferGrowCases {
		t.Run(c.name, func(t *testing.T) {
			b := NewBuffer(5, 10)
			c.setup(b)
			c.grow(b)
			if got := c.checkDim(b); got != c.expected {
				t.Errorf("expected %d %s, got %d", c.expected, c.axis, got)
			}
			if !c.checkPres(b) {
				t.Errorf("expected content preserved and new %s empty", c.axis)
			}
		})
	}
}
