package headlessterm

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// eventScrollback records the order of Push and Pop calls.
type eventScrollback struct {
	testScrollback
	events []string
}

func (s *eventScrollback) Push(line []Cell) {
	s.events = append(s.events, "push")
	s.testScrollback.Push(line)
}

func (s *eventScrollback) Pop() []Cell {
	s.events = append(s.events, "pop")
	return s.testScrollback.Pop()
}

// primaryState is what a grid repaint and a transcript read back from the
// primary screen: every row, the cursor, and the scrollback.
type primaryState struct {
	rows       []string
	cursorRow  int
	cursorCol  int
	scrollback []string
}

func capturePrimary(t *testing.T, term *Terminal, storage *eventScrollback) primaryState {
	t.Helper()
	if term.IsAlternateScreen() {
		t.Fatal("capturePrimary needs the primary screen active")
	}
	var state primaryState
	for row := range term.Rows() {
		state.rows = append(state.rows, term.LineContent(row))
	}
	state.cursorRow, state.cursorCol = term.CursorPos()
	for i := range storage.Len() {
		var b strings.Builder
		for _, cell := range storage.Line(i) {
			b.WriteRune(cell.Char)
		}
		state.scrollback = append(state.scrollback, strings.TrimRight(b.String(), " \x00"))
	}
	return state
}

func diffPrimary(t *testing.T, got, want primaryState) {
	t.Helper()
	if !slices.Equal(got.rows, want.rows) {
		for i := range max(len(got.rows), len(want.rows)) {
			var g, w string
			if i < len(got.rows) {
				g = got.rows[i]
			}
			if i < len(want.rows) {
				w = want.rows[i]
			}
			if g != w {
				t.Errorf("row %d = %q, want %q", i, g, w)
			}
		}
	}
	if got.cursorRow != want.cursorRow || got.cursorCol != want.cursorCol {
		t.Errorf("cursor = (%d,%d), want (%d,%d)", got.cursorRow, got.cursorCol, want.cursorRow, want.cursorCol)
	}
	if !slices.Equal(got.scrollback, want.scrollback) {
		t.Errorf("scrollback = %q, want %q", got.scrollback, want.scrollback)
	}
}

func newResizeTerminal() (*Terminal, *eventScrollback) {
	storage := &eventScrollback{}
	storage.SetMaxLines(1000)
	return New(WithSize(24, 80), WithScrollback(storage)), storage
}

// writeShellOutput fills the primary screen the way a shell session does
// before a fullscreen program starts: 30 output lines, then the command.
func writeShellOutput(term *Terminal) {
	for i := 1; i <= 30; i++ {
		term.WriteString(fmt.Sprintf("output line %02d\r\n", i))
	}
	term.WriteString("$ vim notes.txt\r\n")
}

const (
	enterAlternateScreen = "\x1b[?1049h\x1b[H\x1b[2J~ vim\r\n~\r\n~"
	leaveAlternateScreen = "\x1b[?1049l"
)

// TestResizeOnAlternateScreenKeepsPrimary covers the resize pair a client sends
// around a fullscreen program: shrink after ?1049h, grow after ?1049l.
func TestResizeOnAlternateScreenKeepsPrimary(t *testing.T) {
	control, controlStorage := newResizeTerminal()
	writeShellOutput(control)
	control.WriteString(enterAlternateScreen + leaveAlternateScreen + "$ ")
	want := capturePrimary(t, control, controlStorage)

	term, storage := newResizeTerminal()
	writeShellOutput(term)
	term.WriteString(enterAlternateScreen)
	altBefore := []string{term.LineContent(0), term.LineContent(1), term.LineContent(2)}
	storage.events = nil
	term.Resize(18, 80)
	altAfter := []string{term.LineContent(0), term.LineContent(1), term.LineContent(2)}
	if !slices.Equal(altAfter, altBefore) {
		t.Errorf("alternate rows after shrink = %q, want %q", altAfter, altBefore)
	}
	term.WriteString(leaveAlternateScreen)
	term.Resize(24, 80)
	term.WriteString("$ ")

	diffPrimary(t, capturePrimary(t, term, storage), want)
	assertPushPopPairing(t, storage.events)
}

// TestResizeGrowBeforeLeavingAlternateScreenKeepsPrimary is the same pair with
// the grow arriving while the fullscreen program is still running.
func TestResizeGrowBeforeLeavingAlternateScreenKeepsPrimary(t *testing.T) {
	control, controlStorage := newResizeTerminal()
	writeShellOutput(control)
	control.WriteString(enterAlternateScreen + leaveAlternateScreen + "$ ")
	want := capturePrimary(t, control, controlStorage)

	term, storage := newResizeTerminal()
	writeShellOutput(term)
	term.WriteString(enterAlternateScreen)
	storage.events = nil
	term.Resize(18, 80)
	term.Resize(24, 80)
	term.WriteString(leaveAlternateScreen + "$ ")

	diffPrimary(t, capturePrimary(t, term, storage), want)
	assertPushPopPairing(t, storage.events)
}

// TestResizeOnAlternateScreenWithPrimaryCursorAboveShrink has the primary's
// cursor well above the rows a shrink removes: nothing scrolls, nothing is
// popped back, even though the scrollback holds older lines.
func TestResizeOnAlternateScreenWithPrimaryCursorAboveShrink(t *testing.T) {
	for _, growWhileAlternate := range []bool{false, true} {
		t.Run(fmt.Sprintf("growWhileAlternate=%v", growWhileAlternate), func(t *testing.T) {
			setup := func(term *Terminal) {
				writeShellOutput(term)
				term.WriteString("\x1b[H\x1b[2J$ less log\r\n")
			}
			control, controlStorage := newResizeTerminal()
			setup(control)
			control.WriteString(enterAlternateScreen + leaveAlternateScreen + "$ ")
			want := capturePrimary(t, control, controlStorage)
			if len(want.scrollback) == 0 {
				t.Fatal("setup must leave lines in scrollback for the grow to wrongly pop")
			}

			term, storage := newResizeTerminal()
			setup(term)
			term.WriteString(enterAlternateScreen)
			storage.events = nil
			term.Resize(18, 80)
			if growWhileAlternate {
				term.Resize(24, 80)
				term.WriteString(leaveAlternateScreen)
			} else {
				term.WriteString(leaveAlternateScreen)
				term.Resize(24, 80)
			}
			term.WriteString("$ ")

			diffPrimary(t, capturePrimary(t, term, storage), want)
			if len(storage.events) != 0 {
				t.Errorf("scrollback events = %v, want none", storage.events)
			}
		})
	}
}

// TestResizeOnAlternateScreenPairsLikePrimaryResize checks the scrollback sees
// the same Push/Pop sequence as the identical shrink and grow with the primary
// screen active, which the daemon's transcript tee depends on.
func TestResizeOnAlternateScreenPairsLikePrimaryResize(t *testing.T) {
	primary, primaryStorage := newResizeTerminal()
	writeShellOutput(primary)
	primaryStorage.events = nil
	primary.Resize(18, 80)
	primary.Resize(24, 80)

	alternate, alternateStorage := newResizeTerminal()
	writeShellOutput(alternate)
	alternate.WriteString(enterAlternateScreen)
	alternateStorage.events = nil
	alternate.Resize(18, 80)
	alternate.WriteString(leaveAlternateScreen)
	alternate.Resize(24, 80)

	if !slices.Equal(alternateStorage.events, primaryStorage.events) {
		t.Errorf("events = %v, want %v (primary-active resize)", alternateStorage.events, primaryStorage.events)
	}
}

// assertPushPopPairing requires every pushed line to be popped back, pushes
// first: the shrink scrolls rows out, the grow returns exactly those.
func assertPushPopPairing(t *testing.T, events []string) {
	t.Helper()
	pushes := 0
	for pushes < len(events) && events[pushes] == "push" {
		pushes++
	}
	pops := len(events) - pushes
	for _, e := range events[pushes:] {
		if e != "pop" {
			t.Errorf("events = %v, want pushes then pops", events)
			return
		}
	}
	if pushes == 0 || pushes != pops {
		t.Errorf("events = %v, want N pushes then N pops with N > 0", events)
	}
}

// TestResizeOnAlternateScreenThenPrimaryOutputKeepsPrimary lets the primary
// scroll at the reduced height between leaving the alternate screen and the
// grow. Lines output scrolls off the top use up the rows the shrink cut, so the
// grow must pop those lines back rather than hand back blank rows.
func TestResizeOnAlternateScreenThenPrimaryOutputKeepsPrimary(t *testing.T) {
	for _, lines := range []int{19, 20, 30} { // scrolls 3, 4 and 14 lines at 18 rows
		t.Run(fmt.Sprintf("lines=%d", lines), func(t *testing.T) {
			setup := func(term *Terminal) {
				writeShellOutput(term)
				term.WriteString("\x1b[H\x1b[2J$ less log\r\n")
			}
			output := func(term *Terminal) {
				for i := 1; i <= lines; i++ {
					term.WriteString(fmt.Sprintf("after %02d\r\n", i))
				}
				term.WriteString("$ ")
			}
			control, controlStorage := newResizeTerminal()
			setup(control)
			control.WriteString(enterAlternateScreen + leaveAlternateScreen)
			output(control)
			want := capturePrimary(t, control, controlStorage)

			term, storage := newResizeTerminal()
			setup(term)
			term.WriteString(enterAlternateScreen)
			term.Resize(18, 80)
			term.WriteString(leaveAlternateScreen)
			output(term)
			term.Resize(24, 80)

			diffPrimary(t, capturePrimary(t, term, storage), want)
		})
	}
}

// TestResizeColumnsOnAlternateScreenClampsSavedCursor shrinks the columns while
// the alternate screen is active: the primary resumes on the last column, not
// past the edge of the grid.
func TestResizeColumnsOnAlternateScreenClampsSavedCursor(t *testing.T) {
	term, _ := newResizeTerminal()
	term.WriteString("\x1b[5;71H") // row 4, col 70
	term.WriteString(enterAlternateScreen)
	term.Resize(24, 40)
	term.WriteString(leaveAlternateScreen)

	if row, col := term.CursorPos(); row != 4 || col != 39 {
		t.Fatalf("cursor = (%d,%d), want (4,39)", row, col)
	}
	term.WriteString("X")
	if c := term.Cell(4, 39); c == nil || c.Char != 'X' {
		t.Errorf("cell (4,39) = %v, want 'X'", c)
	}
}

// TestResizeOnAlternateScreenThenHeightIndependentScrollKeepsPrimary runs
// scrolls between leaving the alternate screen and the grow that a terminal
// performs the same at any height: SU, and output at the bottom of a scroll
// region that ends above the last row. They must not use up the rows the
// shrink cut.
func TestResizeOnAlternateScreenThenHeightIndependentScrollKeepsPrimary(t *testing.T) {
	regionOutput := "\x1b[1;10r\x1b[10;1H"
	for i := 1; i <= 5; i++ {
		regionOutput += fmt.Sprintf("region %02d\r\n", i)
	}
	regionOutput += "\x1b[r"
	for name, between := range map[string]string{
		"SU 3":               "\x1b[3S",
		"region 1;10 scroll": regionOutput,
	} {
		t.Run(name, func(t *testing.T) {
			setup := func(term *Terminal) {
				writeShellOutput(term)
				term.WriteString("\x1b[H\x1b[2J$ less log\r\n")
			}
			control, controlStorage := newResizeTerminal()
			setup(control)
			control.WriteString(enterAlternateScreen + leaveAlternateScreen + between + "$ ")
			want := capturePrimary(t, control, controlStorage)

			term, storage := newResizeTerminal()
			setup(term)
			term.WriteString(enterAlternateScreen)
			term.Resize(18, 80)
			term.WriteString(leaveAlternateScreen + between)
			term.Resize(24, 80)
			term.WriteString("$ ")

			diffPrimary(t, capturePrimary(t, term, storage), want)
		})
	}
}
