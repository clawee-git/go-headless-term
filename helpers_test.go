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
