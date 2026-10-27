package headlessterm

type WrapScrollbackProvider interface {
	ScrollbackProvider
	PushWrapped(line []Cell, wrapped bool)
	LineWrapped(index int) bool
	PopWrapped() ([]Cell, bool)
	ReplaceWrapped(lines [][]Cell, wrapped []bool)
}

func (m *MemoryScrollback) trimToMax() {
	if m.maxLines > 0 && len(m.lines) > m.maxLines {
		excess := len(m.lines) - m.maxLines
		m.lines = m.lines[excess:]
		m.wrapped = m.wrapped[excess:]
	}
}

func (m *MemoryScrollback) PushWrapped(line []Cell, wrapped bool) {
	m.Push(line)
	m.wrapped[len(m.wrapped)-1] = wrapped
}

func (m *MemoryScrollback) LineWrapped(index int) bool {
	if index < 0 || index >= len(m.wrapped) {
		return false
	}
	return m.wrapped[index]
}

func (m *MemoryScrollback) PopWrapped() ([]Cell, bool) {
	if len(m.lines) == 0 {
		return nil, false
	}
	wrapped := m.wrapped[len(m.wrapped)-1]
	return m.Pop(), wrapped
}

func (m *MemoryScrollback) ReplaceWrapped(lines [][]Cell, wrapped []bool) {
	m.lines = make([][]Cell, len(lines))
	m.wrapped = make([]bool, len(lines))
	for i, line := range lines {
		m.lines[i] = append([]Cell(nil), line...)
		m.wrapped[i] = i < len(wrapped) && wrapped[i]
	}
	m.trimToMax()
}

func pushScrollbackLine(storage ScrollbackProvider, line []Cell, wrapped bool) {
	if w, ok := storage.(WrapScrollbackProvider); ok {
		w.PushWrapped(line, wrapped)
		return
	}
	storage.Push(line)
}
