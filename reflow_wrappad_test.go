package headlessterm

import "testing"

func TestIsWrapPadOnLiveWideWrap(t *testing.T) {
	term := newReflowTerm(3, 10, NewMemoryScrollback(100))
	term.WriteString("abcdefghi你")
	if !term.Cell(0, 9).IsWrapPad() {
		t.Fatal("padding cell before a wrapped wide character reads false")
	}
	if term.Cell(0, 8).IsWrapPad() {
		t.Fatal("a letter reads as padding")
	}
}

func TestIsWrapPadAfterReflowLeavesEdgeBlank(t *testing.T) {
	term := newReflowTerm(4, 20, NewMemoryScrollback(100))
	term.WriteString("abcdefghijklmnopqrs你")
	term.Resize(4, 10)
	if !term.Cell(1, 9).IsWrapPad() {
		t.Fatal("layout padding before a wide character reads false")
	}
}

func TestIsWrapPadFalseForTypedSpace(t *testing.T) {
	term := newReflowTerm(3, 10, NewMemoryScrollback(100))
	term.WriteString("abcdefghi 你")
	if term.Cell(0, 9).IsWrapPad() {
		t.Fatal("a typed space reads as padding")
	}
}

func TestIsWrapPadNeverTrueWithReflowOff(t *testing.T) {
	term := New(WithSize(3, 10), WithScrollback(NewMemoryScrollback(100)))
	term.WriteString("abcdefghi你")
	for col := 0; col < 10; col++ {
		if term.Cell(0, col).IsWrapPad() {
			t.Fatalf("col %d reads as padding with reflow off", col)
		}
	}
}
