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

func TestResizeOnAlternateScreenKeepsPrimary(t *testing.T) {
	t.Run("resize pair around fullscreen", resizePairAroundFullscreenCase)
	t.Run("pairs like primary resize", resizePairsLikePrimaryCase)
	for _, c := range alternateResizeCases {
		t.Run(c.name, func(t *testing.T) { runAlternateResizeCase(t, c) })
	}
}

// alternateResizeCase varies one fixed procedure by data alone: fill the
// primary screen, enter the alternate screen, shrink, write something, and grow
// back — with the grow landing either side of leaving the alternate screen. The
// primary screen must come back exactly as a control terminal that was never
// resized leaves it.
type alternateResizeCase struct {
	name               string
	clearBeforeAlt     bool   // setup ends on a fresh prompt, with older lines in scrollback
	growWhileAlternate bool   // the grow arrives before ?1049l rather than after
	between            string // written after leaving the alternate screen, before the grow
	after              string // written after the grow
	wantEvents         string // "paired", "none", or "" for unchecked
}

var alternateResizeCases = []alternateResizeCase{
	{name: "grow before leaving", growWhileAlternate: true, after: "$ ", wantEvents: "paired"},
	{name: "primary cursor above shrink, grow after leaving", clearBeforeAlt: true, after: "$ ", wantEvents: "none"},
	{name: "primary cursor above shrink, grow while alternate", clearBeforeAlt: true, growWhileAlternate: true, after: "$ ", wantEvents: "none"},
	{name: "primary output before grow, lines=19", clearBeforeAlt: true, between: afterLines(19)},
	{name: "primary output before grow, lines=20", clearBeforeAlt: true, between: afterLines(20)},
	{name: "primary output before grow, lines=30", clearBeforeAlt: true, between: afterLines(30)},
	{name: "height independent scroll, SU 3", clearBeforeAlt: true, between: "\x1b[3S", after: "$ "},
	{name: "height independent scroll, region 1;10 scroll", clearBeforeAlt: true, between: regionScrollOutput(5), after: "$ "},
}

// afterLines is the shell output a case writes at the reduced height: n lines
// and a prompt. At 18 rows, 19, 20 and 30 lines scroll 3, 4 and 14 lines off.
func afterLines(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "after %02d\r\n", i)
	}
	b.WriteString("$ ")
	return b.String()
}

// regionScrollOutput is n lines written at the bottom of a 1;10 scroll region,
// a scroll a terminal performs the same way at any height.
func regionScrollOutput(n int) string {
	var b strings.Builder
	b.WriteString("\x1b[1;10r\x1b[10;1H")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "region %02d\r\n", i)
	}
	b.WriteString("\x1b[r")
	return b.String()
}

func runAlternateResizeCase(t *testing.T, c alternateResizeCase) {
	setup := func(term *Terminal) {
		writeShellOutput(term)
		if c.clearBeforeAlt {
			term.WriteString("\x1b[H\x1b[2J$ less log\r\n")
		}
	}

	control, controlStorage := newResizeTerminal()
	setup(control)
	control.WriteString(enterAlternateScreen + leaveAlternateScreen + c.between + c.after)
	want := capturePrimary(t, control, controlStorage)
	if c.wantEvents == "none" && len(want.scrollback) == 0 {
		t.Fatal("setup must leave lines in scrollback for the grow to wrongly pop")
	}

	term, storage := newResizeTerminal()
	setup(term)
	term.WriteString(enterAlternateScreen)
	storage.events = nil
	term.Resize(18, 80)
	if c.growWhileAlternate {
		term.Resize(24, 80)
		term.WriteString(leaveAlternateScreen + c.between + c.after)
	} else {
		term.WriteString(leaveAlternateScreen + c.between)
		term.Resize(24, 80)
		term.WriteString(c.after)
	}

	diffPrimary(t, capturePrimary(t, term, storage), want)
	switch c.wantEvents {
	case "paired":
		assertPushPopPairing(t, storage.events)
	case "none":
		if len(storage.events) != 0 {
			t.Errorf("scrollback events = %v, want none", storage.events)
		}
	}
}

// resizePairAroundFullscreenCase covers the resize pair a client sends
// around a fullscreen program: shrink after ?1049h, grow after ?1049l.
func resizePairAroundFullscreenCase(t *testing.T) {
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

// resizePairsLikePrimaryCase checks the scrollback sees the same Push/Pop
// sequence as the identical shrink and grow with the primary screen active,
// which the daemon's transcript tee depends on.
func resizePairsLikePrimaryCase(t *testing.T) {
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
