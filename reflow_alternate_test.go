package headlessterm

import "testing"

func TestAlternateNotReflowed(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(6, 80, sb)
	term.WriteString("\x1b[?1049h" + digits(60))
	term.Resize(6, 40)
	if got := term.LineContent(0); got != digits(60)[:40] {
		t.Fatalf("alternate row 0 %q want truncated", got)
	}
	if term.LineContent(1) != "" {
		t.Fatalf("alternate row 1 %q want empty", term.LineContent(1))
	}
}

func TestPrimaryReflowedUnderAlternate(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(6, 80, sb)
	term.WriteString(digits(100) + "\x1b[?1049h")
	term.Resize(6, 40)
	term.WriteString("\x1b[?1049l")
	rows := screenText(term)
	if rows[0] != digits(100)[:40] || rows[1] != digits(100)[40:80] || rows[2] != digits(100)[80:] {
		t.Fatalf("primary rows %q", rows[:3])
	}
	if row, col := term.CursorPos(); row != 2 || col != 20 {
		t.Fatalf("cursor %d,%d want 2,20", row, col)
	}
}
