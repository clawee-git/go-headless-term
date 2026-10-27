package headlessterm

import (
	"slices"
	"strings"
	"testing"
)

func TestPushWrappedKeepsFlag(t *testing.T) {
	m := NewMemoryScrollback(10)
	line := []Cell{NewCell()}
	m.PushWrapped(line, true)
	m.Push(line)
	m.PushWrapped(line, true)
	if got := []bool{m.LineWrapped(0), m.LineWrapped(1), m.LineWrapped(2), m.LineWrapped(3)}; !slices.Equal(got, []bool{true, false, true, false}) {
		t.Fatalf("flags %v", got)
	}
	if _, w := m.PopWrapped(); !w {
		t.Fatal("PopWrapped lost the flag")
	}
	m.Pop()
	m.PushWrapped(line, true)
	if !m.LineWrapped(1) || m.LineWrapped(0) != true {
		t.Fatal("flags misaligned after Pop")
	}
}

func TestScrollUpPushesFlags(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(3, 80, sb)
	term.WriteString(digits(100) + "\r\na\r\nb\r\nc\r\nd")
	if sb.pushes != 0 || sb.pushWrapped == 0 {
		t.Fatalf("pushes %d pushWrapped %d", sb.pushes, sb.pushWrapped)
	}
	if !sb.LineWrapped(0) || sb.LineWrapped(1) {
		t.Fatalf("flags %v %v", sb.LineWrapped(0), sb.LineWrapped(1))
	}
}

func allText(term *Terminal, sb ScrollbackProvider) string {
	var lines []string
	for i := range sb.Len() {
		lines = append(lines, strings.TrimRight(cellsText([][]Cell{sb.Line(i)})[0], " "))
	}
	return strings.Join(append(lines, screenText(term)...), "\n")
}

func TestRejoinAcrossScrollbackBoundary(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(3, 80, sb)
	term.WriteString(digits(100) + "\r\na\r\nb")
	if sb.Len() == 0 {
		t.Fatal("fixture should push the first half into scrollback")
	}
	term.Resize(3, 120)
	if !strings.Contains(allText(term, sb), digits(100)+"\n") {
		t.Fatalf("line not rejoined:\n%s", allText(term, sb))
	}
}

func TestProviderWithoutWrapReadsNotWrapped(t *testing.T) {
	sb := &testScrollback{}
	sb.SetMaxLines(100)
	term := newReflowTerm(3, 80, sb)
	term.WriteString(digits(100) + "\r\na\r\nb\r\nc")
	kept := sb.Len()
	term.Resize(3, 40)
	if sb.Len() != kept {
		t.Fatalf("scrollback changed %d -> %d", kept, sb.Len())
	}
}

func TestReflowTrimsToMaxLines(t *testing.T) {
	sb := newWrapCounting(5)
	term := newReflowTerm(3, 80, sb)
	for range 12 {
		term.WriteString(digits(70) + "\r\n")
	}
	term.Resize(3, 20)
	if sb.Len() != 5 {
		t.Fatalf("scrollback %d want 5", sb.Len())
	}
}

func TestReflowReplacesScrollbackOnce(t *testing.T) {
	sb := newWrapCounting(100)
	term := newReflowTerm(3, 80, sb)
	for range 6 {
		term.WriteString(digits(100) + "\r\n")
	}
	p, pw, o, ow := sb.pushes, sb.pushWrapped, sb.pops, sb.popWrapped
	term.Resize(3, 40)
	if sb.replaces != 1 || sb.pushes != p || sb.pushWrapped != pw || sb.pops != o || sb.popWrapped != ow {
		t.Fatalf("replaces %d push %d/%d pop %d/%d", sb.replaces, sb.pushes-p, sb.pushWrapped-pw, sb.pops-o, sb.popWrapped-ow)
	}
}

func TestProviderWithoutReplaceKeepsScrollback(t *testing.T) {
	sb := &testScrollback{}
	sb.SetMaxLines(100)
	term := newReflowTerm(3, 80, sb)
	for range 6 {
		term.WriteString(digits(100) + "\r\n")
	}
	before := cellsText(captureReflow(term, sb).history)
	term.Resize(3, 40)
	if got := cellsText(captureReflow(term, sb).history); !slices.Equal(got, before) {
		t.Fatalf("scrollback changed:\n%q\n%q", before, got)
	}
	if got := term.LineContent(0); got != digits(100)[:40] && got != digits(100)[40:80] && got != digits(100)[80:] {
		t.Fatalf("screen not reflowed: %q", got)
	}
}

func TestProviderWithoutReplaceKeepsEveryLine(t *testing.T) {
	sb := &testScrollback{}
	sb.SetMaxLines(100)
	term := newReflowTerm(3, 80, sb)
	lines := []string{"A" + digits(78) + "z", "B" + digits(78) + "z", "C" + digits(78) + "z"}
	term.WriteString(lines[0] + "\r\n" + lines[1] + "\r\n" + lines[2])
	term.Resize(3, 40)
	got := strings.Join(append(cellsText(sb.lines), screenText(term)...), "")
	if want := strings.Join(lines, ""); got != want {
		t.Fatalf("lines lost after narrowing:\n%q\n%q", want, got)
	}
	if sb.pushCount != 3 {
		t.Fatalf("pushCount %d, want 3 (append-only)", sb.pushCount)
	}
}
