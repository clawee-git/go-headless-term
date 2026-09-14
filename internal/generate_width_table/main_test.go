package main

import (
	"bytes"
	"errors"
	"go/format"
	"slices"
	"strings"
	"testing"
)

func TestParseProperty(t *testing.T) {
	data := `# EastAsianWidth-16.0.0.txt
# @missing: 0000..10FFFF; N

0020           ; Na # Zs         SPACE
1100..115F     ; W  # Lo    [96] HANGUL CHOSEONG KIYEOK..HANGUL CHOSEONG FILLER
23FA           ; N  # So         BLACK CIRCLE FOR RECORD
FF01..FF60     ; F  # Po    [96] FULLWIDTH EXCLAMATION MARK..FULLWIDTH WHITE PARENTHESIS
`
	got, err := parseProperty(strings.NewReader(data), isWideEastAsian)
	if err != nil {
		t.Fatal(err)
	}
	want := []span{{0x1100, 0x115F}, {0xFF01, 0xFF60}}
	if !slices.Equal(got, want) {
		t.Errorf("got %X, want %X", got, want)
	}
}

func TestParsePropertyRejectsMalformedLines(t *testing.T) {
	for _, line := range []string{
		"1100 W",         // no separator
		"11G0 ; W",       // not hex
		"115F..1100 ; W", // reversed range
		"1100.. ; W",     // missing end
		"110000000 ; W",  // beyond 32 bits
	} {
		_, err := parseProperty(strings.NewReader(line), isWideEastAsian)
		if !errors.Is(err, errBadLine) {
			t.Errorf("%q: err = %v, want errBadLine", line, err)
		}
	}
}

func TestCheckVersion(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"ucd header", "# EastAsianWidth-16.0.0.txt\n# Date: x\n0020 ; Na\n", false},
		{"emoji header", "# emoji-data.txt\n# Used with Emoji Version 16.0 and subsequent minor revisions\n231A ; Emoji_Presentation\n", false},
		{"other ucd version", "# EastAsianWidth-15.1.0.txt\n0020 ; Na\n", true},
		{"other emoji version", "# emoji-data.txt\n# Used with Emoji Version 15.1 and later\n", true},
		{"version only in data comment", "# EastAsianWidth.txt\n0020 ; Na # EastAsianWidth-16.0.0.txt\n", true},
	}
	for _, tt := range tests {
		err := checkVersion([]byte(tt.data), "16.0.0")
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestMergeSpans(t *testing.T) {
	got := mergeSpans([]span{{10, 12}, {1, 3}, {4, 5}, {11, 20}, {30, 30}})
	want := []span{{1, 5}, {10, 20}, {30, 30}}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := mergeSpans(nil); len(got) != 0 {
		t.Errorf("mergeSpans(nil) = %v, want empty", got)
	}
}

func TestSubtractSpans(t *testing.T) {
	a := []span{{0, 9}, {20, 29}, {40, 49}}
	b := []span{{0, 2}, {5, 5}, {9, 21}, {45, 60}}
	got := subtractSpans(a, b)
	want := []span{{3, 4}, {6, 8}, {22, 29}, {40, 44}}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRenderCompiles checks the rendered table is valid Go and that a span
// crossing U+FFFF is split between R16 and R32 without losing a code point.
func TestRenderCompiles(t *testing.T) {
	src, err := render("16.0.0", []span{{0x00, 0x1F}, {0x300, 0x36F}}, []span{{0xFF01, 0x10010}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := format.Source(src); err != nil {
		t.Fatalf("rendered source does not parse: %v", err)
	}
	for _, want := range []string{
		"{Lo: 0xFF01, Hi: 0xFFFF, Stride: 1}",
		"{Lo: 0x10000, Hi: 0x10010, Stride: 1}",
		"LatinOffset: 1,",
		`const widthTableUnicodeVersion = "16.0.0"`,
	} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("rendered source lacks %q", want)
		}
	}
}
