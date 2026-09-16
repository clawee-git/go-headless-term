package headlessterm

import (
	"os"
	"regexp"
	"testing"
	"unicode"
)

func TestWidthFunctions(t *testing.T) {
	t.Run("runeWidth", func(t *testing.T) {
		for _, tt := range []struct {
			r        rune
			expected int
		}{
			{'A', 1},
			{'a', 1},
			{'1', 1},
			{' ', 1},
			{'中', 2},
			{'日', 2},
			{'本', 2},
			{'한', 2},
			{'글', 2},
			{'가', 2},
			{'Ａ', 2},
			{0, 0},
			{'Ａ', 2},
			{'\U0001F600', 2},
			{'✅', 2},
			{'⌛', 2},
			{'⏺', 1},
			{'✻', 1},
			{'✢', 1},
			{'✳', 1},
			{'✶', 1},
			{'✽', 1},
			{'❯', 1},
			{'●', 1},
			{'…', 1},
			{'└', 1},
			{'⎿', 1},
			{'́', 0},
			{'​', 0},
			{'゙', 0},
			{'⁠', 0},
			{'⁦', 0},
			{'\U000E0067', 0},
			{0x7F, 0},
		} {
			got := runeWidth(tt.r)
			if got != tt.expected {
				t.Errorf("runeWidth(%q) = %d, want %d", tt.r, got, tt.expected)
			}
		}
	})

	t.Run("isWideRune", func(t *testing.T) {
		for _, tt := range []struct {
			r        rune
			expected bool
		}{
			{'A', false},
			{'a', false},
			{' ', false},
			{'中', true},
			{'日', true},
			{'한', true},
			{'가', true},
			{'Ａ', true},
			{'0', false},
		} {
			got := isWideRune(tt.r)
			if got != tt.expected {
				t.Errorf("isWideRune(%q) = %v, want %v", tt.r, got, tt.expected)
			}
		}
	})

	t.Run("StringWidth", func(t *testing.T) {
		for _, tt := range []struct {
			s        string
			expected int
		}{
			{"Hello", 5},
			{"中文", 4},
			{"Hello中文", 9},
			{"", 0},
			{"한글", 4},
			{"⏺ Bash", 6},
			{"❯ hi", 4},
			{"é", 1},
			{"\U0001F3F4\U000E0067\U000E0062\U000E0065\U000E006E\U000E0067\U000E007F", 2},
		} {
			got := StringWidth(tt.s)
			if got != tt.expected {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, got, tt.expected)
			}
		}
	})
}

// TestWidthFunctionsAgree checks that runeWidth, isWideRune and StringWidth
// answer from the same table for every scalar value.
func TestWidthFunctionsAgree(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue // surrogates are not scalar values; string(r) would be U+FFFD
		}
		w := runeWidth(r)
		if isWideRune(r) != (w == 2) {
			t.Fatalf("isWideRune(%U) = %v, runeWidth = %d", r, isWideRune(r), w)
		}
		if sw := StringWidth(string(r)); sw != w {
			t.Fatalf("StringWidth(%U) = %d, runeWidth = %d", r, sw, w)
		}
	}
}

// TestWidthTableMatchesGenerateDirective fails when width_table.go was
// generated for a different Unicode version than width.go's go:generate
// directive asks for.
func TestWidthTableMatchesGenerateDirective(t *testing.T) {
	src, err := os.ReadFile("width.go")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^//go:generate go run \./internal/generate_width_table -unicode (\S+) -out width_table\.go$`).FindSubmatch(src)
	if m == nil {
		t.Fatal("width.go has no generate_width_table directive")
	}
	if got := string(m[1]); got != widthTableUnicodeVersion {
		t.Errorf("directive asks for Unicode %s, width_table.go is %s", got, widthTableUnicodeVersion)
	}
}
