package headlessterm

// testScrollback is a test implementation of ScrollbackProvider shared by the
// package's tests.
type testScrollback struct {
	lines     [][]Cell
	maxLines  int
	pushCount int
}

func (s *testScrollback) Push(line []Cell) {
	s.pushCount++
	lineCopy := make([]Cell, len(line))
	copy(lineCopy, line)
	s.lines = append(s.lines, lineCopy)
	if s.maxLines > 0 && len(s.lines) > s.maxLines {
		s.lines = s.lines[len(s.lines)-s.maxLines:]
	}
}

func (s *testScrollback) Len() int { return len(s.lines) }

func (s *testScrollback) Line(index int) []Cell {
	if index < 0 || index >= len(s.lines) {
		return nil
	}
	return s.lines[index]
}

func (s *testScrollback) Clear()              { s.lines = make([][]Cell, 0) }
func (s *testScrollback) SetMaxLines(max int) { s.maxLines = max }
func (s *testScrollback) MaxLines() int       { return s.maxLines }

func (s *testScrollback) Pop() []Cell {
	if len(s.lines) == 0 {
		return nil
	}
	line := s.lines[len(s.lines)-1]
	s.lines = s.lines[:len(s.lines)-1]
	return line
}

// newScrollbackTerm builds a terminal of rows x 80 backed by a fresh
// testScrollback holding up to 100 lines: the fixture every resize-and-scrollback
// test starts from.
func newScrollbackTerm(rows int, opts ...Option) (*Terminal, *testScrollback) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)
	all := append([]Option{WithSize(rows, 80)}, opts...)
	all = append(all, WithScrollback(storage))
	return New(all...), storage
}

// writeNumberedLines writes prefix0 .. prefix<n-1>, one per line, leaving the
// cursor on the last line (no trailing newline).
func writeNumberedLines(term *Terminal, prefix string, n int) {
	for i := 0; i < n; i++ {
		if i > 0 {
			term.WriteString("\r\n")
		}
		term.WriteString(prefix + string(rune('0'+i)))
	}
}
