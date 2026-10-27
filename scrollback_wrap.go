package headlessterm

type WrapScrollbackProvider interface {
	ScrollbackProvider
	PushWrapped(line []Cell, wrapped bool)
	LineWrapped(index int) bool
	PopWrapped() ([]Cell, bool)
	ReplaceWrapped(lines [][]Cell, wrapped []bool)
}

func (m *MemoryScrollback) PushWrapped(line []Cell, wrapped bool) { m.Push(line) }

func (m *MemoryScrollback) LineWrapped(index int) bool { return false }

func (m *MemoryScrollback) PopWrapped() ([]Cell, bool) { return m.Pop(), false }

func (m *MemoryScrollback) ReplaceWrapped(lines [][]Cell, wrapped []bool) {}
