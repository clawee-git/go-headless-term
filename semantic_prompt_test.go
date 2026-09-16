package headlessterm

import (
	"testing"

	"github.com/danielgatis/go-ansicode"
)

var semanticPromptMarkTypeCases = []struct {
	name      string
	osc       string
	wantType  ansicode.ShellIntegrationMark
	wantExit  int
	checkExit bool
}{
	{"prompt start", "\x1b]133;A\x07", ansicode.PromptStart, -1, true},
	{"command start", "\x1b]133;B\x07", ansicode.CommandStart, -1, false},
	{"command executed", "\x1b]133;C\x07", ansicode.CommandExecuted, -1, false},
	{"command finished", "\x1b]133;D\x07", ansicode.CommandFinished, -1, true},
	{"finished exit 0", "\x1b]133;D;0\x07", ansicode.CommandFinished, 0, true},
	{"finished exit 1", "\x1b]133;D;1\x07", ansicode.CommandFinished, 1, true},
	{"finished exit 127", "\x1b]133;D;127\x07", ansicode.CommandFinished, 127, true},
}

func TestSemanticPromptMark_Types(t *testing.T) {
	for _, c := range semanticPromptMarkTypeCases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(24, 80))
			term.WriteString(c.osc)

			marks := term.PromptMarks()
			if len(marks) != 1 {
				t.Fatalf("expected 1 mark, got %d", len(marks))
			}
			if marks[0].Type != c.wantType {
				t.Errorf("expected mark type %d, got %d", c.wantType, marks[0].Type)
			}
			if c.checkExit && marks[0].ExitCode != c.wantExit {
				t.Errorf("expected exit code %d, got %d", c.wantExit, marks[0].ExitCode)
			}
		})
	}
}

func TestSemanticPromptMark_FullSequence(t *testing.T) {
	term := New(WithSize(24, 80))

	// Simulate a full shell prompt cycle
	term.WriteString("\x1b]133;A\x07")     // Prompt start
	term.WriteString("$ ")                 // Prompt text
	term.WriteString("\x1b]133;B\x07")     // Command start
	term.WriteString("ls -la")             // User input
	term.WriteString("\r\n")               // Enter
	term.WriteString("\x1b]133;C\x07")     // Command executed
	term.WriteString("file1\r\nfile2\r\n") // Command output
	term.WriteString("\x1b]133;D;0\x07")   // Command finished with exit code 0

	marks := term.PromptMarks()
	if len(marks) != 4 {
		t.Fatalf("expected 4 marks, got %d", len(marks))
	}

	// Check mark types in order
	expected := []ansicode.ShellIntegrationMark{
		ansicode.PromptStart,
		ansicode.CommandStart,
		ansicode.CommandExecuted,
		ansicode.CommandFinished,
	}

	for i, exp := range expected {
		if marks[i].Type != exp {
			t.Errorf("mark %d: expected type %d, got %d", i, exp, marks[i].Type)
		}
	}

	// Check exit code of the last mark
	if marks[3].ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", marks[3].ExitCode)
	}
}

func TestSemanticPromptMark_RowTracking(t *testing.T) {
	term := New(WithSize(24, 80))

	// Add marks at different rows
	term.WriteString("\x1b]133;A\x07") // Row 0
	term.WriteString("prompt1\r\n")
	term.WriteString("\x1b]133;A\x07") // Row 1
	term.WriteString("prompt2\r\n")
	term.WriteString("\x1b]133;A\x07") // Row 2

	marks := term.PromptMarks()
	if len(marks) != 3 {
		t.Fatalf("expected 3 marks, got %d", len(marks))
	}

	// Rows should be tracked correctly
	if marks[0].Row != 0 {
		t.Errorf("mark 0: expected row 0, got %d", marks[0].Row)
	}
	if marks[1].Row != 1 {
		t.Errorf("mark 1: expected row 1, got %d", marks[1].Row)
	}
	if marks[2].Row != 2 {
		t.Errorf("mark 2: expected row 2, got %d", marks[2].Row)
	}
}

func TestSemanticPromptMark_NextPromptRow(t *testing.T) {
	term := New(WithSize(24, 80))

	// Add prompts at different absolute rows
	term.WriteString("\x1b]133;A\x07") // Absolute row 0
	term.WriteString("prompt1\r\n")
	term.WriteString("\x1b]133;A\x07") // Absolute row 1
	term.WriteString("prompt2\r\n")
	term.WriteString("\x1b]133;A\x07") // Absolute row 2

	// Find next prompt from absolute row -1 (before any content)
	next := term.NextPromptRow(-1, -1)
	if next != 0 {
		t.Errorf("expected next prompt at absolute row 0, got %d", next)
	}

	// Find next prompt from absolute row 0
	next = term.NextPromptRow(0, -1)
	if next != 1 {
		t.Errorf("expected next prompt at absolute row 1, got %d", next)
	}

	// Find next prompt from absolute row 1
	next = term.NextPromptRow(1, -1)
	if next != 2 {
		t.Errorf("expected next prompt at absolute row 2, got %d", next)
	}

	// No next prompt from absolute row 2
	next = term.NextPromptRow(2, -1)
	if next != -1 {
		t.Errorf("expected no next prompt (-1), got %d", next)
	}
}

func TestSemanticPromptMark_PrevPromptRow(t *testing.T) {
	term := New(WithSize(24, 80))

	// Add prompts at different absolute rows
	term.WriteString("\x1b]133;A\x07") // Absolute row 0
	term.WriteString("prompt1\r\n")
	term.WriteString("\x1b]133;A\x07") // Absolute row 1
	term.WriteString("prompt2\r\n")
	term.WriteString("\x1b]133;A\x07") // Absolute row 2

	// Find previous prompt from absolute row 3
	prev := term.PrevPromptRow(3, -1)
	if prev != 2 {
		t.Errorf("expected prev prompt at absolute row 2, got %d", prev)
	}

	// Find previous prompt from absolute row 2
	prev = term.PrevPromptRow(2, -1)
	if prev != 1 {
		t.Errorf("expected prev prompt at absolute row 1, got %d", prev)
	}

	// Find previous prompt from absolute row 1
	prev = term.PrevPromptRow(1, -1)
	if prev != 0 {
		t.Errorf("expected prev prompt at absolute row 0, got %d", prev)
	}

	// No previous prompt from absolute row 0
	prev = term.PrevPromptRow(0, -1)
	if prev != -1 {
		t.Errorf("expected no prev prompt (-1), got %d", prev)
	}
}

func TestSemanticPromptMark_FilterByType(t *testing.T) {
	term := New(WithSize(24, 80))

	// Add different mark types at absolute rows
	term.WriteString("\x1b]133;A\x07") // PromptStart at absolute row 0
	term.WriteString("prompt\r\n")
	term.WriteString("\x1b]133;B\x07") // CommandStart at absolute row 1
	term.WriteString("cmd\r\n")
	term.WriteString("\x1b]133;C\x07") // CommandExecuted at absolute row 2
	term.WriteString("output\r\n")
	term.WriteString("\x1b]133;A\x07") // PromptStart at absolute row 3

	// Find next PromptStart only using absolute rows
	next := term.NextPromptRow(-1, ansicode.PromptStart)
	if next != 0 {
		t.Errorf("expected next PromptStart at absolute row 0, got %d", next)
	}

	next = term.NextPromptRow(0, ansicode.PromptStart)
	if next != 3 {
		t.Errorf("expected next PromptStart at absolute row 3, got %d", next)
	}
}

func TestSemanticPromptMark_ClearMarks(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]133;A\x07")
	term.WriteString("\x1b]133;B\x07")

	if term.PromptMarkCount() != 2 {
		t.Fatalf("expected 2 marks, got %d", term.PromptMarkCount())
	}

	term.ClearPromptMarks()

	if term.PromptMarkCount() != 0 {
		t.Errorf("expected 0 marks after clear, got %d", term.PromptMarkCount())
	}
}

func TestSemanticPromptMark_GetMarkAt(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]133;A\x07") // Absolute row 0

	// Get mark at absolute row 0
	mark := term.GetPromptMarkAt(0)
	if mark == nil {
		t.Fatal("expected mark at absolute row 0, got nil")
	}
	if mark.Type != ansicode.PromptStart {
		t.Errorf("expected PromptStart, got %d", mark.Type)
	}

	// No mark at absolute row 1
	mark = term.GetPromptMarkAt(1)
	if mark != nil {
		t.Errorf("expected nil at absolute row 1, got %v", mark)
	}
}

type testSemanticPromptHandler struct {
	marks []ansicode.ShellIntegrationMark
	codes []int
}

func (p *testSemanticPromptHandler) OnMark(mark ansicode.ShellIntegrationMark, exitCode int) {
	p.marks = append(p.marks, mark)
	p.codes = append(p.codes, exitCode)
}

func TestSemanticPromptMark_Handler(t *testing.T) {
	t.Run("receives marks", func(t *testing.T) {
		handler := &testSemanticPromptHandler{}
		term := New(WithSize(24, 80), WithSemanticPromptHandler(handler))

		term.WriteString("\x1b]133;A\x07")
		term.WriteString("\x1b]133;D;42\x07")

		if len(handler.marks) != 2 {
			t.Fatalf("expected handler to receive 2 marks, got %d", len(handler.marks))
		}

		if handler.marks[0] != ansicode.PromptStart {
			t.Errorf("expected PromptStart, got %d", handler.marks[0])
		}
		if handler.marks[1] != ansicode.CommandFinished {
			t.Errorf("expected CommandFinished, got %d", handler.marks[1])
		}
		if handler.codes[1] != 42 {
			t.Errorf("expected exit code 42, got %d", handler.codes[1])
		}
	})

	t.Run("st terminator", func(t *testing.T) {
		term := New(WithSize(24, 80))

		// OSC 133 ; A ST (using ESC \ as string terminator)
		term.WriteString("\x1b]133;A\x1b\\")

		marks := term.PromptMarks()
		if len(marks) != 1 {
			t.Fatalf("expected 1 mark, got %d", len(marks))
		}

		if marks[0].Type != ansicode.PromptStart {
			t.Errorf("expected PromptStart mark, got %d", marks[0].Type)
		}
	})
}

func TestSemanticPromptMark_Middleware(t *testing.T) {
	var middlewareCalled bool
	var receivedMark ansicode.ShellIntegrationMark
	var receivedExitCode int

	mw := &Middleware{
		SemanticPromptMark: func(mark ansicode.ShellIntegrationMark, exitCode int, next func(ansicode.ShellIntegrationMark, int)) {
			middlewareCalled = true
			receivedMark = mark
			receivedExitCode = exitCode
			next(mark, exitCode)
		},
	}

	term := New(WithSize(24, 80), WithMiddleware(mw))

	term.WriteString("\x1b]133;D;123\x07")

	if !middlewareCalled {
		t.Error("expected middleware to be called")
	}
	if receivedMark != ansicode.CommandFinished {
		t.Errorf("expected CommandFinished, got %d", receivedMark)
	}
	if receivedExitCode != 123 {
		t.Errorf("expected exit code 123, got %d", receivedExitCode)
	}

	// Verify the mark was still stored
	if term.PromptMarkCount() != 1 {
		t.Errorf("expected 1 mark, got %d", term.PromptMarkCount())
	}
}

// --- GetLastCommandOutput Tests ---

var getLastCommandOutputCases = []struct {
	name   string
	writes []string
	want   string
}{
	{
		name: "basic",
		writes: []string{
			"\x1b]133;A\x07", "$ ", "\x1b]133;B\x07", "echo hello", "\r\n",
			"\x1b]133;C\x07", "hello\r\n", "\x1b]133;D;0\x07",
		},
		want: "hello",
	},
	{
		name: "multi line",
		writes: []string{
			"\x1b]133;C\x07", "line1\r\n", "line2\r\n", "line3\r\n", "\x1b]133;D;0\x07",
		},
		want: "line1\nline2\nline3",
	},
	{
		name:   "no output",
		writes: []string{"\x1b]133;C\x07", "\x1b]133;D;0\x07"},
		want:   "",
	},
	{
		name:   "no marks",
		writes: nil,
		want:   "",
	},
	{
		name:   "only executed no finished",
		writes: []string{"\x1b]133;C\x07", "output\r\n"},
		want:   "",
	},
	{
		name: "multiple commands",
		writes: []string{
			"\x1b]133;C\x07", "first output\r\n", "\x1b]133;D;0\x07",
			"\x1b]133;A\x07", "$ ", "\x1b]133;B\x07", "cmd2\r\n",
			"\x1b]133;C\x07", "second output\r\n", "\x1b]133;D;0\x07",
		},
		want: "second output",
	},
	{
		name:   "with exit code",
		writes: []string{"\x1b]133;C\x07", "error message\r\n", "\x1b]133;D;1\x07"},
		want:   "error message",
	},
	{
		name: "trailing empty lines",
		writes: []string{
			"\x1b]133;C\x07", "content\r\n", "\r\n", "\r\n", "\x1b]133;D;0\x07",
		},
		want: "content",
	},
}

func TestGetLastCommandOutput_Cases(t *testing.T) {
	for _, c := range getLastCommandOutputCases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(24, 80))
			for _, w := range c.writes {
				term.WriteString(w)
			}

			if output := term.GetLastCommandOutput(); output != c.want {
				t.Errorf("expected %q, got %q", c.want, output)
			}
		})
	}
}

// --- Scrollback Tests for Absolute Row Functions ---

type testScrollbackForSemanticPrompt struct {
	lines    [][]Cell
	maxLines int
}

func (s *testScrollbackForSemanticPrompt) Push(line []Cell) {
	lineCopy := make([]Cell, len(line))
	copy(lineCopy, line)
	s.lines = append(s.lines, lineCopy)
	if s.maxLines > 0 && len(s.lines) > s.maxLines {
		s.lines = s.lines[len(s.lines)-s.maxLines:]
	}
}

func (s *testScrollbackForSemanticPrompt) Len() int {
	return len(s.lines)
}

func (s *testScrollbackForSemanticPrompt) Line(index int) []Cell {
	if index < 0 || index >= len(s.lines) {
		return nil
	}
	return s.lines[index]
}

func (s *testScrollbackForSemanticPrompt) SetMaxLines(n int) {
	s.maxLines = n
}

func (s *testScrollbackForSemanticPrompt) Clear() {
	s.lines = nil
}

func (s *testScrollbackForSemanticPrompt) MaxLines() int {
	return s.maxLines
}

func (s *testScrollbackForSemanticPrompt) Pop() []Cell {
	if len(s.lines) == 0 {
		return nil
	}
	line := s.lines[len(s.lines)-1]
	s.lines = s.lines[:len(s.lines)-1]
	return line
}

// newScrollbackPromptTerm returns a 5-row terminal with scrollback and a
// prompt mark at absolute row 0, ready for navigation tests.
func newScrollbackPromptTerm() *Terminal {
	storage := &testScrollbackForSemanticPrompt{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(5, 80), WithScrollback(storage))

	term.WriteString("\x1b]133;A\x07")
	term.WriteString("prompt1\r\n")

	return term
}

func TestSemanticPromptMark_ScrollbackNavigation(t *testing.T) {
	t.Run("next prompt row", semanticPromptNextWithScrollback)
	t.Run("prev prompt row", semanticPromptPrevWithScrollback)
	t.Run("get mark at", semanticPromptGetMarkAtWithScrollback)
}

func semanticPromptNextWithScrollback(t *testing.T) {
	term := newScrollbackPromptTerm()

	// Write enough lines to push content into scrollback
	for i := 0; i < 10; i++ {
		term.WriteString("line\r\n")
	}

	// Add another prompt (this will be at a higher absolute row)
	term.WriteString("\x1b]133;A\x07")
	term.WriteString("prompt2\r\n")

	marks := term.PromptMarks()
	if len(marks) != 2 {
		t.Fatalf("expected 2 marks, got %d", len(marks))
	}

	// First mark should be at absolute row 0
	if marks[0].Row != 0 {
		t.Errorf("expected first mark at absolute row 0, got %d", marks[0].Row)
	}

	// Second mark should be at absolute row 11 (0 + 1 + 10 lines)
	if marks[1].Row != 11 {
		t.Errorf("expected second mark at absolute row 11, got %d", marks[1].Row)
	}

	// NextPromptRow should return absolute rows
	next := term.NextPromptRow(-1, -1)
	if next != 0 {
		t.Errorf("expected next prompt at absolute row 0, got %d", next)
	}

	next = term.NextPromptRow(0, -1)
	if next != 11 {
		t.Errorf("expected next prompt at absolute row 11, got %d", next)
	}

	// Verify scrollback exists
	scrollbackLen := term.ScrollbackLen()
	if scrollbackLen == 0 {
		t.Error("expected scrollback to exist")
	}
}

func semanticPromptPrevWithScrollback(t *testing.T) {
	term := newScrollbackPromptTerm()

	// Write enough lines to push content into scrollback
	for i := 0; i < 10; i++ {
		term.WriteString("line\r\n")
	}

	// Add another prompt
	term.WriteString("\x1b]133;A\x07")

	marks := term.PromptMarks()

	// PrevPromptRow should return absolute rows
	prev := term.PrevPromptRow(marks[1].Row+1, -1)
	if prev != marks[1].Row {
		t.Errorf("expected prev prompt at absolute row %d, got %d", marks[1].Row, prev)
	}

	prev = term.PrevPromptRow(marks[1].Row, -1)
	if prev != 0 {
		t.Errorf("expected prev prompt at absolute row 0, got %d", prev)
	}

	prev = term.PrevPromptRow(0, -1)
	if prev != -1 {
		t.Errorf("expected no prev prompt (-1), got %d", prev)
	}
}

func semanticPromptGetMarkAtWithScrollback(t *testing.T) {
	term := newScrollbackPromptTerm()

	// Write enough lines to push the prompt into scrollback
	for i := 0; i < 10; i++ {
		term.WriteString("line\r\n")
	}

	// GetPromptMarkAt should find mark at absolute row 0 even when in scrollback
	mark := term.GetPromptMarkAt(0)
	if mark == nil {
		t.Fatal("expected mark at absolute row 0, got nil")
	}
	if mark.Type != ansicode.PromptStart {
		t.Errorf("expected PromptStart, got %d", mark.Type)
	}

	// No mark at absolute row 5
	mark = term.GetPromptMarkAt(5)
	if mark != nil {
		t.Errorf("expected nil at absolute row 5, got %v", mark)
	}
}
