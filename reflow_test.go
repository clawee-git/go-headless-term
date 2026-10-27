package headlessterm

import (
	"slices"
	"strings"
	"testing"
)

func writeHistory(term *Terminal, wrappedLen int, last string) {
	for i := range 8 {
		n := 20
		if i == 3 {
			n = wrappedLen
		}
		term.WriteString("row" + string(rune('0'+i)) + " " + digits(n) + "\r\n")
	}
	term.WriteString(last)
}

var roundTripCases = []struct {
	name  string
	write func(*Terminal)
	row   int
	col   int
}{
	{"plain", func(t *Terminal) { writeHistory(t, 130, "tail "+digits(30)) }, 5, 35},
	{"sgr", func(t *Terminal) {
		writeHistory(t, 100, "\x1b[31mred\x1b[0m plain \x1b[1;44mbold\x1b[0m "+digits(60))
	}, 5, 76},
	{"wide at edge", func(t *Terminal) { writeHistory(t, 90, digits(79)+"你z") }, 6, 2},
	{"cursor mid line", func(t *Terminal) { writeHistory(t, 130, digits(30)+"\x1b[10G") }, 5, 9},
	{"cursor at end", func(t *Terminal) { writeHistory(t, 130, digits(79)) }, 5, 79},
	{"pending wrap", func(t *Terminal) { writeHistory(t, 130, digits(80)) }, 5, 80},
}

func TestNarrowWidenRoundTrip(t *testing.T) {
	for _, tc := range roundTripCases {
		t.Run(tc.name, func(t *testing.T) {
			sb := newWrapCounting(100)
			term := newReflowTerm(6, 80, sb)
			tc.write(term)
			before := captureReflow(term, sb)
			term.Resize(6, 40)
			term.Resize(6, 80)
			requireSameReflowState(t, captureReflow(term, sb), before)
		})
	}
}

func TestReflowOffIsTodaysResize(t *testing.T) {
	plain := newWrapCounting(100)
	off := newWrapCounting(100)
	a := New(WithSize(6, 80), WithScrollback(plain))
	b := New(WithSize(6, 80), WithScrollback(off), WithReflow(false))
	for _, term := range []*Terminal{a, b} {
		writeHistory(term, 130, digits(30))
		term.Resize(6, 40)
	}
	requireSameReflowState(t, captureReflow(b, off), captureReflow(a, plain))
	if got := a.LineContent(5); got != digits(30) {
		t.Fatalf("narrowing without reflow should keep top-left cells, got %q", got)
	}
	if off.replaces != 0 {
		t.Fatalf("reflow off must not replace scrollback, got %d", off.replaces)
	}
}

func TestWrappedCommandRewraps(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(6, 80, sb)
	cmd := digits(100)
	term.WriteString(cmd)
	term.Resize(6, 40)
	if got, want := wrapFlags(term)[:4], []bool{true, true, false, false}; !slices.Equal(got, want) {
		t.Fatalf("flags at 40: got %v want %v", got, want)
	}
	if got := strings.Join(screenText(term)[:3], ""); got != cmd {
		t.Fatalf("text across the run: got %q want %q", got, cmd)
	}
	term.Resize(6, 120)
	if got, want := wrapFlags(term)[:2], []bool{false, false}; !slices.Equal(got, want) {
		t.Fatalf("flags at 120: got %v want %v", got, want)
	}
	if got := term.LineContent(0); got != cmd || term.LineContent(1) != "" {
		t.Fatalf("at 120 the command should be one row, got %q / %q", got, term.LineContent(1))
	}
}

func TestHardLineWrapsAndKeepsBreak(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(6, 80, sb)
	term.WriteString(digits(60) + "\r\nnext")
	term.Resize(6, 40)
	rows := screenText(term)
	if rows[0] != digits(60)[:40] || rows[1] != digits(60)[40:] || rows[2] != "next" {
		t.Fatalf("rows: %q", rows[:4])
	}
	if got, want := wrapFlags(term)[:3], []bool{true, false, false}; !slices.Equal(got, want) {
		t.Fatalf("flags: got %v want %v", got, want)
	}
}

func TestCursorFollowsItsCell(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(6, 80, sb)
	text := digits(100)
	term.WriteString(text + "\x1b[2;6H")
	term.Resize(6, 40)
	row, col := term.CursorPos()
	if row != 2 || col != 5 {
		t.Fatalf("cursor at %d,%d want 2,5", row, col)
	}
	if got := term.Cell(row, col).Char; got != rune(text[85]) {
		t.Fatalf("cursor cell %q want %q", got, rune(text[85]))
	}
}

func TestSavedCursorFollows(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(6, 80, sb)
	term.WriteString(digits(100) + "\x1b[2;6H\x1b7\x1b[1;1H")
	term.Resize(6, 40)
	term.WriteString("\x1b8")
	if row, col := term.CursorPos(); row != 2 || col != 5 {
		t.Fatalf("restored cursor at %d,%d want 2,5", row, col)
	}
}

func TestRowAndColumnChangeTogether(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(6, 80, sb)
	writeHistory(term, 130, "tail")
	pushes, pops := sb.pushes, sb.pops
	term.Resize(4, 40)
	if term.Rows() != 4 || term.Cols() != 40 {
		t.Fatalf("size %dx%d", term.Rows(), term.Cols())
	}
	if sb.replaces != 1 || sb.pushes != pushes || sb.pops != pops {
		t.Fatalf("replaces %d pushes %d->%d pops %d->%d", sb.replaces, pushes, sb.pushes, pops, sb.pops)
	}
	if got := screenText(term)[3]; got != "tail" {
		t.Fatalf("last row %q want tail", got)
	}
	if row, col := term.CursorPos(); row != 3 || col != 4 {
		t.Fatalf("cursor %d,%d want 3,4", row, col)
	}
}

func TestRowGrowRestoresWrapFlags(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(3, 80, sb)
	term.WriteString(digits(100) + "\r\na\r\nb")
	term.Resize(5, 80)
	if !term.IsWrapped(0) {
		t.Fatal("popped scrollback line lost its wrap flag")
	}
}
