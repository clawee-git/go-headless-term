package headlessterm

import "unicode"

//go:generate go run ./internal/generate_width_table -unicode 16.0.0 -out width_table.go

// runeWidth returns the number of columns a printed rune occupies, answered
// from the tables in width_table.go (see widthTableUnicodeVersion):
//   - 0 for controls (Cc), combining marks (Mn, Me, Mc), U+00AD, U+200B..U+200F
//     and U+FEFF;
//   - 2 for East_Asian_Width W or F, and Emoji_Presentation=Yes;
//   - 1 for everything else, East Asian ambiguous width included.
//
// Text-presentation pictographs such as U+23FA (⏺) and U+276F (❯) are one
// column, as terminals draw them and as the applications positioning the
// cursor around them count them.
func runeWidth(r rune) int {
	if r >= 0x20 && r < 0x7F {
		return 1
	}
	if unicode.Is(zeroWidthTable, r) {
		return 0
	}
	if unicode.Is(wideTable, r) {
		return 2
	}
	return 1
}

// isWideRune returns true if the rune occupies 2 columns (CJK ideographs, fullwidth forms, emoji-presentation emoji).
func isWideRune(r rune) bool {
	return runeWidth(r) == 2
}

// StringWidth returns the total display width of a string: the sum of each
// rune's width, exactly as the terminal lays the runes out in cells.
func StringWidth(s string) int {
	width := 0
	for _, r := range s {
		width += runeWidth(r)
	}
	return width
}
