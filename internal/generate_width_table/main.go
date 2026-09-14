// Command generate_width_table writes the rune width tables the emulator
// sizes printed runes with, from one version of the Unicode Character
// Database. It is run by `go generate` from the module root (see width.go):
//
//	go run ./internal/generate_width_table -unicode 16.0.0 -out width_table.go
//
// The rule it encodes, in priority order:
//
//   - width 0: General_Category Cc, Mn, Me or Mc, plus U+00AD, U+200B..U+200F
//     and U+FEFF;
//   - width 2: East_Asian_Width W or F, or Emoji_Presentation=Yes;
//   - width 1: everything else, East_Asian_Width A (ambiguous) included.
//
// Text-presentation pictographs such as U+23FA or U+276F are neither East
// Asian Wide nor emoji-presentation, so they stay one column, which is how
// terminals and the applications drawing on them size them.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// span is an inclusive range of code points.
type span struct {
	lo, hi rune
}

// extraZeroWidth lists the format characters (General_Category Cf) that are
// zero width in addition to the categories above: soft hyphen, the zero-width
// space/joiners and directional marks, and the byte order mark.
var extraZeroWidth = []span{
	{0x00AD, 0x00AD},
	{0x200B, 0x200F},
	{0xFEFF, 0xFEFF},
}

var errBadLine = errors.New("malformed data line")

func main() {
	version := flag.String("unicode", "", "Unicode Character Database version, e.g. 16.0.0")
	out := flag.String("out", "", "Go file to write")
	base := flag.String("base", "https://www.unicode.org/Public", "Unicode data root URL")
	flag.Parse()
	if *version == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*base, *version, *out); err != nil {
		fmt.Fprintln(os.Stderr, "generate_width_table:", err)
		os.Exit(1)
	}
}

func run(base, version, out string) error {
	zero, wide, err := loadTables(base, version)
	if err != nil {
		return err
	}
	src, err := render(version, zero, wide)
	if err != nil {
		return err
	}
	return os.WriteFile(out, src, 0o666)
}

// loadTables fetches the three data files and derives the zero-width and wide
// sets. A rune in both is zero width, so it is removed from the wide set.
func loadTables(base, version string) (zero, wide []span, err error) {
	root := strings.TrimSuffix(base, "/") + "/" + version + "/ucd/"
	categories, err := loadProperty(root+"extracted/DerivedGeneralCategory.txt", version, isZeroWidthCategory)
	if err != nil {
		return nil, nil, err
	}
	eastAsian, err := loadProperty(root+"EastAsianWidth.txt", version, isWideEastAsian)
	if err != nil {
		return nil, nil, err
	}
	emoji, err := loadProperty(root+"emoji/emoji-data.txt", version, isEmojiPresentation)
	if err != nil {
		return nil, nil, err
	}
	zero = mergeSpans(slices.Concat(categories, extraZeroWidth))
	wide = subtractSpans(mergeSpans(slices.Concat(eastAsian, emoji)), zero)
	return zero, wide, nil
}

func isZeroWidthCategory(value string) bool {
	switch value {
	case "Cc", "Mn", "Me", "Mc":
		return true
	}
	return false
}

func isWideEastAsian(value string) bool { return value == "W" || value == "F" }

func isEmojiPresentation(value string) bool { return value == "Emoji_Presentation" }

// loadProperty fetches one UCD file, checks that it is the requested version,
// and returns the ranges whose property value satisfies keep.
func loadProperty(url, version string, keep func(string) bool) ([]span, error) {
	data, err := fetch(url)
	if err != nil {
		return nil, err
	}
	if err := checkVersion(data, version); err != nil {
		return nil, fmt.Errorf("%s: %w", url, err)
	}
	spans, err := parseProperty(bytes.NewReader(data), keep)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", url, err)
	}
	return spans, nil
}

func fetch(url string) ([]byte, error) {
	client := &http.Client{Timeout: time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// checkVersion refuses a file whose header names another version, so a URL
// or mirror serving different data cannot silently change the tables. UCD
// files open with "# <Name>-<version>.txt"; emoji-data.txt instead says
// "Used with Emoji Version <major>.<minor>".
func checkVersion(data []byte, version string) error {
	var header []byte
	for line := range bytes.Lines(data) {
		if !bytes.HasPrefix(line, []byte("#")) && len(bytes.TrimSpace(line)) > 0 {
			break
		}
		header = append(header, line...)
	}
	major, minor, _ := strings.Cut(version, ".")
	minor, _, _ = strings.Cut(minor, ".")
	emojiVersion := "Emoji Version " + major + "." + minor + " "
	if !bytes.Contains(header, []byte("-"+version+".txt")) && !bytes.Contains(header, []byte(emojiVersion)) {
		return fmt.Errorf("header does not name version %s", version)
	}
	return nil
}

// parseProperty reads "XXXX[..YYYY] ; Value # comment" lines.
func parseProperty(r io.Reader, keep func(string) bool) ([]span, error) {
	var spans []span
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line, _, _ := strings.Cut(scanner.Text(), "#")
		if strings.TrimSpace(line) == "" {
			continue
		}
		field, value, ok := strings.Cut(line, ";")
		if !ok {
			return nil, fmt.Errorf("%w: %q", errBadLine, scanner.Text())
		}
		if !keep(strings.TrimSpace(value)) {
			continue
		}
		s, err := parseSpan(strings.TrimSpace(field))
		if err != nil {
			return nil, err
		}
		spans = append(spans, s)
	}
	return spans, scanner.Err()
}

func parseSpan(field string) (span, error) {
	loText, hiText, isRange := strings.Cut(field, "..")
	if !isRange {
		hiText = loText
	}
	lo, err := strconv.ParseUint(loText, 16, 32)
	if err != nil {
		return span{}, fmt.Errorf("%w: code point %q", errBadLine, field)
	}
	hi, err := strconv.ParseUint(hiText, 16, 32)
	if err != nil || hi < lo {
		return span{}, fmt.Errorf("%w: code point %q", errBadLine, field)
	}
	return span{rune(lo), rune(hi)}, nil
}

// mergeSpans sorts spans and joins any that overlap or touch.
func mergeSpans(spans []span) []span {
	sorted := slices.Clone(spans)
	slices.SortFunc(sorted, func(a, b span) int { return int(a.lo - b.lo) })
	var merged []span
	for _, s := range sorted {
		if n := len(merged); n > 0 && s.lo <= merged[n-1].hi+1 {
			merged[n-1].hi = max(merged[n-1].hi, s.hi)
			continue
		}
		merged = append(merged, s)
	}
	return merged
}

// subtractSpans returns the parts of a not covered by b. Both must be merged.
func subtractSpans(a, b []span) []span {
	var result []span
	j := 0
	for _, s := range a {
		for j < len(b) && b[j].hi < s.lo {
			j++
		}
		lo := s.lo
		for k := j; k < len(b) && b[k].lo <= s.hi; k++ {
			if b[k].lo > lo {
				result = append(result, span{lo, b[k].lo - 1})
			}
			lo = max(lo, b[k].hi+1)
		}
		if lo <= s.hi {
			result = append(result, span{lo, s.hi})
		}
	}
	return result
}

func render(version string, zero, wide []span) ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by internal/generate_width_table from Unicode %s; DO NOT EDIT.\n", version)
	buf.WriteString("//\n// Regenerate from the module root with:\n//\n")
	fmt.Fprintf(&buf, "//\tgo run ./internal/generate_width_table -unicode %s -out width_table.go\n\n", version)
	buf.WriteString("package headlessterm\n\nimport \"unicode\"\n\n")
	buf.WriteString("// widthTableUnicodeVersion is the Unicode Character Database version the\n// width tables below were generated from.\n")
	fmt.Fprintf(&buf, "const widthTableUnicodeVersion = %q\n\n", version)
	buf.WriteString("// zeroWidthTable holds General_Category Cc, Mn, Me and Mc, plus U+00AD,\n// U+200B..U+200F and U+FEFF.\n")
	writeRangeTable(&buf, "zeroWidthTable", zero)
	buf.WriteString("// wideTable holds East_Asian_Width W and F, and Emoji_Presentation=Yes,\n// minus anything in zeroWidthTable.\n")
	writeRangeTable(&buf, "wideTable", wide)
	return format.Source(buf.Bytes())
}

// writeRangeTable renders spans as a unicode.RangeTable, splitting a span that
// crosses U+FFFF between R16 and R32.
func writeRangeTable(buf *bytes.Buffer, name string, spans []span) {
	var r16, r32 []span
	for _, s := range spans {
		switch {
		case s.hi <= 0xFFFF:
			r16 = append(r16, s)
		case s.lo > 0xFFFF:
			r32 = append(r32, s)
		default:
			r16 = append(r16, span{s.lo, 0xFFFF})
			r32 = append(r32, span{0x10000, s.hi})
		}
	}
	latinOffset := 0
	for _, s := range r16 {
		if s.hi <= 0xFF {
			latinOffset++
		}
	}
	fmt.Fprintf(buf, "var %s = &unicode.RangeTable{\n\tR16: []unicode.Range16{\n", name)
	for _, s := range r16 {
		fmt.Fprintf(buf, "\t\t{Lo: 0x%04X, Hi: 0x%04X, Stride: 1},\n", s.lo, s.hi)
	}
	buf.WriteString("\t},\n\tR32: []unicode.Range32{\n")
	for _, s := range r32 {
		fmt.Fprintf(buf, "\t\t{Lo: 0x%05X, Hi: 0x%05X, Stride: 1},\n", s.lo, s.hi)
	}
	fmt.Fprintf(buf, "\t},\n\tLatinOffset: %d,\n}\n\n", latinOffset)
}
