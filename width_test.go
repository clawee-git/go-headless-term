package headlessterm

import (
	"os"
	"regexp"
	"testing"
	"unicode"
)

func TestRuneWidth(t *testing.T) {
	tests := []struct {
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
		{'Ａ', 2}, // Fullwidth A
		{0, 0},

		// East_Asian_Width W or F, and Emoji_Presentation=Yes: two columns.
		{'\uFF21', 2},     // Ａ FULLWIDTH LATIN CAPITAL LETTER A
		{'\U0001F600', 2}, // 😀 GRINNING FACE
		{'\u2705', 2},     // ✅ WHITE HEAVY CHECK MARK
		{'\u231B', 2},     // ⌛ HOURGLASS

		// Text-presentation pictographs terminals draw in one column.
		{'\u23FA', 1}, // ⏺ BLACK CIRCLE FOR RECORD
		{'\u273B', 1}, // ✻ TEARDROP-SPOKED ASTERISK
		{'\u2722', 1}, // ✢ FOUR TEARDROP-SPOKED ASTERISK
		{'\u2733', 1}, // ✳ EIGHT SPOKED ASTERISK
		{'\u2736', 1}, // ✶ SIX POINTED BLACK STAR
		{'\u273D', 1}, // ✽ HEAVY TEARDROP-SPOKED ASTERISK
		{'\u276F', 1}, // ❯ HEAVY RIGHT-POINTING ANGLE QUOTATION MARK ORNAMENT

		// East_Asian_Width A (ambiguous) and N: one column.
		{'\u25CF', 1}, // ● BLACK CIRCLE
		{'\u2026', 1}, // … HORIZONTAL ELLIPSIS
		{'\u2514', 1}, // └ BOX DRAWINGS LIGHT UP AND RIGHT
		{'\u23BF', 1}, // ⎿ BOTTOM LEFT CORNER

		// Zero width.
		{'\u0301', 0},     // COMBINING ACUTE ACCENT
		{'\u200B', 0},     // ZERO WIDTH SPACE
		{'\u3099', 0},     // COMBINING KATAKANA-HIRAGANA VOICED SOUND MARK: Mn outranks East_Asian_Width W
		{'\u2060', 0},     // WORD JOINER (Cf)
		{'\u2066', 0},     // LEFT-TO-RIGHT ISOLATE (Cf)
		{'\U000E0067', 0}, // TAG LATIN SMALL LETTER G (Cf)
		{0x7F, 0},         // DELETE
	}

	for _, tt := range tests {
		got := runeWidth(tt.r)
		if got != tt.expected {
			t.Errorf("runeWidth(%q) = %d, want %d", tt.r, got, tt.expected)
		}
	}
}

func TestIsWideRune(t *testing.T) {
	tests := []struct {
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
		{'Ａ', true}, // Fullwidth A
		{'0', false},
	}

	for _, tt := range tests {
		got := isWideRune(tt.r)
		if got != tt.expected {
			t.Errorf("isWideRune(%q) = %v, want %v", tt.r, got, tt.expected)
		}
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		s        string
		expected int
	}{
		{"Hello", 5},
		{"中文", 4},
		{"Hello中文", 9},
		{"", 0},
		{"한글", 4},
		{"\u23FA Bash", 6},
		{"\u276F hi", 4},
		{"e\u0301", 1},
		{"\U0001F3F4\U000E0067\U000E0062\U000E0065\U000E006E\U000E0067\U000E007F", 2}, // England flag: black flag + tag sequence
	}

	for _, tt := range tests {
		got := StringWidth(tt.s)
		if got != tt.expected {
			t.Errorf("StringWidth(%q) = %d, want %d", tt.s, got, tt.expected)
		}
	}
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
