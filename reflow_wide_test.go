package headlessterm

import "testing"

func TestWidePadDroppedOnRejoin(t *testing.T) {
	term := newReflowTerm(3, 10, NewMemoryScrollback(100))
	term.WriteString("abcdefghi你")
	term.Resize(3, 20)
	if got := term.LineContent(0); got != "abcdefghi你" {
		t.Fatalf("pad not dropped: %q", got)
	}
}

func TestRealSpaceBeforeWideSurvivesRejoin(t *testing.T) {
	term := newReflowTerm(3, 10, NewMemoryScrollback(100))
	term.WriteString("abcdefghi 你")
	term.Resize(3, 20)
	if got := term.LineContent(0); got != "abcdefghi 你" {
		t.Fatalf("typed space lost: %q", got)
	}
}

func TestLayoutPadDroppedOnWiden(t *testing.T) {
	term := newReflowTerm(4, 20, NewMemoryScrollback(100))
	term.WriteString("abcdefghijklmnopqrs你")
	term.Resize(4, 10)
	term.Resize(4, 22)
	if got := term.LineContent(0); got != "abcdefghijklmnopqrs你" {
		t.Fatalf("layout pad not dropped: %q", got)
	}
}
