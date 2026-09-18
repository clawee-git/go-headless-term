package headlessterm

import (
	"encoding/base64"
	"image/color"
	"testing"
)

// snapshotTextCases: a text snapshot carries the line text and nothing else.
var snapshotTextCases = []struct {
	name      string
	writes    []string
	wantLines []string
}{
	{name: "content", writes: []string{"Hello", "\x1b[2;1H", "World"}, wantLines: []string{"Hello", "World", ""}},
	{name: "empty terminal", wantLines: []string{"", "", ""}},
}

func TestSnapshot_Text(t *testing.T) {
	for _, c := range snapshotTextCases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(3, 10))
			for _, w := range c.writes {
				term.WriteString(w)
			}

			snap := term.Snapshot(SnapshotDetailText)

			if snap.Size.Rows != 3 || snap.Size.Cols != 10 {
				t.Errorf("Size = %dx%d, want 3x10", snap.Size.Rows, snap.Size.Cols)
			}
			if len(snap.Lines) != len(c.wantLines) {
				t.Fatalf("len(Lines) = %d, want %d", len(snap.Lines), len(c.wantLines))
			}
			for i, want := range c.wantLines {
				if snap.Lines[i].Text != want {
					t.Errorf("Lines[%d].Text = %q, want %q", i, snap.Lines[i].Text, want)
				}
				if snap.Lines[i].Segments != nil {
					t.Errorf("Lines[%d]: text mode must not carry segments", i)
				}
				if snap.Lines[i].Cells != nil {
					t.Errorf("Lines[%d]: text mode must not carry cells", i)
				}
			}
		})
	}
	t.Run("cursor", snapshotTextCursorCase)
}

func snapshotTextCursorCase(t *testing.T) {
	term := New(WithSize(5, 10))
	term.WriteString("ABC")

	snap := term.Snapshot(SnapshotDetailText)

	if snap.Cursor.Row != 0 || snap.Cursor.Col != 3 {
		t.Errorf("Cursor = (%d,%d), want (0,3)", snap.Cursor.Row, snap.Cursor.Col)
	}
	if !snap.Cursor.Visible {
		t.Error("Cursor.Visible = false, want true")
	}
	if snap.Cursor.Style != "block" {
		t.Errorf("Cursor.Style = %q, want %q", snap.Cursor.Style, "block")
	}
}

func TestSnapshot_Styled(t *testing.T) {
	t.Run("segments", func(t *testing.T) {
		term := New(WithSize(3, 20))
		term.WriteString("\x1b[31mRed\x1b[0m Normal \x1b[32mGreen\x1b[0m")

		snap := term.Snapshot(SnapshotDetailStyled)
		if len(snap.Lines) < 1 {
			t.Fatal("Expected at least 1 line")
		}
		line := snap.Lines[0]
		if len(line.Segments) < 3 {
			t.Fatalf("Expected at least 3 segments, got %d", len(line.Segments))
		}
		if line.Segments[0].Text != "Red" {
			t.Errorf("Segment[0].Text = %q, want %q", line.Segments[0].Text, "Red")
		}
		if line.Cells != nil {
			t.Error("Styled mode should not have cells")
		}
	})

	t.Run("segment merging", func(t *testing.T) {
		term := New(WithSize(3, 30))
		term.WriteString("\x1b[31mRedText\x1b[0m")

		snap := term.Snapshot(SnapshotDetailStyled)
		if len(snap.Lines[0].Segments) < 1 {
			t.Fatal("Expected at least 1 segment")
		}
		if snap.Lines[0].Segments[0].Text != "RedText" {
			t.Errorf("Segment[0].Text = %q, want %q", snap.Lines[0].Segments[0].Text, "RedText")
		}
	})
}

func TestSnapshot_Full(t *testing.T) {
	t.Run("cells", func(t *testing.T) {
		term := New(WithSize(3, 10))
		term.WriteString("Hi")

		snap := term.Snapshot(SnapshotDetailFull)
		if len(snap.Lines) < 1 {
			t.Fatal("Expected at least 1 line")
		}
		line := snap.Lines[0]
		if len(line.Cells) != 10 {
			t.Fatalf("Expected 10 cells, got %d", len(line.Cells))
		}
		if line.Cells[0].Char != "H" || line.Cells[1].Char != "i" || line.Cells[2].Char != " " {
			t.Errorf("Cells = %q/%q/%q, want H/i/space", line.Cells[0].Char, line.Cells[1].Char, line.Cells[2].Char)
		}
	})

	t.Run("bold attribute", func(t *testing.T) {
		term := New(WithSize(3, 20))
		term.WriteString("\x1b[1mBold\x1b[0m")

		snap := term.Snapshot(SnapshotDetailFull)
		if len(snap.Lines[0].Cells) < 4 {
			t.Fatal("Expected at least 4 cells")
		}
		for i := 0; i < 4; i++ {
			if !snap.Lines[0].Cells[i].Attributes.Bold {
				t.Errorf("Cell[%d] should be bold", i)
			}
		}
	})
}

func TestSnapshot_Styles(t *testing.T) {
	cases := []struct {
		name     string
		sequence string
		field    string
		expected string
	}{
		{"underline single", "\x1b[4mText\x1b[0m", "underline", "single"},
		{"underline single_4:1", "\x1b[4:1mText\x1b[0m", "underline", "single"},
		{"underline double", "\x1b[4:2mText\x1b[0m", "underline", "double"},
		{"underline curly", "\x1b[4:3mText\x1b[0m", "underline", "curly"},
		{"underline dotted", "\x1b[4:4mText\x1b[0m", "underline", "dotted"},
		{"underline dashed", "\x1b[4:5mText\x1b[0m", "underline", "dashed"},
		{"blink slow", "\x1b[5mText\x1b[0m", "blink", "slow"},
		{"blink fast", "\x1b[6mText\x1b[0m", "blink", "fast"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(3, 20))
			term.WriteString(c.sequence)

			snap := term.Snapshot(SnapshotDetailFull)
			if len(snap.Lines[0].Cells) < 4 {
				t.Fatal("Expected at least 4 cells")
			}

			cell := snap.Lines[0].Cells[0]
			var got string
			if c.field == "underline" {
				got = cell.Attributes.Underline
			} else {
				got = cell.Attributes.Blink
			}
			if got != c.expected {
				t.Errorf("%s = %q, want %q", c.field, got, c.expected)
			}
		})
	}

	t.Run("underline color", func(t *testing.T) {
		term := New(WithSize(3, 20))
		term.WriteString("\x1b[4m\x1b[58;2;255;0;0mText\x1b[0m")

		snap := term.Snapshot(SnapshotDetailFull)
		if len(snap.Lines[0].Cells) < 4 {
			t.Fatal("Expected at least 4 cells")
		}

		got := snap.Lines[0].Cells[0].UnderlineColor
		if got != "#ff0000" {
			t.Errorf("UnderlineColor = %q, want %q", got, "#ff0000")
		}
	})
}

func TestSnapshot_Hyperlink(t *testing.T) {
	term := New(WithSize(3, 40))
	term.WriteString("\x1b]8;id=test;https://example.com\x07Link\x1b]8;;\x07")

	snap := term.Snapshot(SnapshotDetailFull)
	if len(snap.Lines[0].Cells) < 4 {
		t.Fatal("Expected at least 4 cells")
	}

	for i := 0; i < 4; i++ {
		cell := snap.Lines[0].Cells[i]
		if cell.Hyperlink == nil {
			t.Errorf("Cell[%d] should have hyperlink", i)
			continue
		}
		if cell.Hyperlink.URI != "https://example.com" {
			t.Errorf("Cell[%d].Hyperlink.URI = %q, want %q", i, cell.Hyperlink.URI, "https://example.com")
		}
	}
}

func TestSnapshot_WideChar(t *testing.T) {
	term := New(WithSize(3, 10))
	term.WriteString("中")

	snap := term.Snapshot(SnapshotDetailFull)
	if len(snap.Lines[0].Cells) < 2 {
		t.Fatal("Expected at least 2 cells")
	}
	if !snap.Lines[0].Cells[0].Wide {
		t.Error("Cell[0] should be wide")
	}
	if !snap.Lines[0].Cells[1].WideSpacer {
		t.Error("Cell[1] should be wide spacer")
	}
}

func TestColorToHex(t *testing.T) {
	tests := []struct {
		name     string
		color    color.Color
		expected string
	}{
		{"nil", nil, ""},
		{"black", color.RGBA{0, 0, 0, 255}, "#000000"},
		{"white", color.RGBA{255, 255, 255, 255}, "#ffffff"},
		{"red", color.RGBA{255, 0, 0, 255}, "#ff0000"},
		{"indexed", &IndexedColor{Index: 1}, "#cd3131"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorToHex(tt.color)
			if result != tt.expected {
				t.Errorf("colorToHex(%v) = %q, want %q", tt.color, result, tt.expected)
			}
		})
	}
}

func TestCursorStyleToString(t *testing.T) {
	tests := []struct {
		style    CursorStyle
		expected string
	}{
		{CursorStyleBlinkingBlock, "block"},
		{CursorStyleSteadyBlock, "block"},
		{CursorStyleBlinkingUnderline, "underline"},
		{CursorStyleSteadyUnderline, "underline"},
		{CursorStyleBlinkingBar, "bar"},
		{CursorStyleSteadyBar, "bar"},
	}

	for _, tt := range tests {
		result := cursorStyleToString(tt.style)
		if result != tt.expected {
			t.Errorf("cursorStyleToString(%v) = %q, want %q", tt.style, result, tt.expected)
		}
	}
}

var snapshotImageCases = []struct {
	name      string
	setup     func(*Terminal) uint32
	wantCount int
}{
	{
		name: "with image",
		setup: func(term *Terminal) uint32 {
			imgData := []byte{
				255, 0, 0, 255,
				0, 255, 0, 255,
				0, 0, 255, 255,
				255, 255, 0, 255,
			}
			imgID := term.images.Store(2, 2, imgData)
			term.images.Place(&ImagePlacement{
				ImageID: imgID,
				Row:     1,
				Col:     2,
				Rows:    3,
				Cols:    4,
				ZIndex:  0,
			})
			return imgID
		},
		wantCount: 1,
	},
	{
		name: "no image",
		setup: func(term *Terminal) uint32 {
			term.WriteString("Hello")
			return 0
		},
		wantCount: 0,
	},
}

func TestSnapshot_Images(t *testing.T) {
	for _, c := range snapshotImageCases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(10, 20))
			imgID := c.setup(term)
			snap := term.Snapshot(SnapshotDetailText)

			if len(snap.Images) != c.wantCount {
				t.Fatalf("Expected %d images, got %d", c.wantCount, len(snap.Images))
			}
			if c.wantCount == 0 {
				return
			}

			img := snap.Images[0]
			if img.ID != imgID {
				t.Errorf("Image.ID = %d, want %d", img.ID, imgID)
			}
			if img.Row != 1 || img.Col != 2 || img.Rows != 3 || img.Cols != 4 {
				t.Errorf("Image placement = (%d,%d %dx%d), want (1,2 3x4)", img.Row, img.Col, img.Rows, img.Cols)
			}
			if img.PixelWidth != 2 || img.PixelHeight != 2 {
				t.Errorf("Image pixel size = %dx%d, want 2x2", img.PixelWidth, img.PixelHeight)
			}
		})
	}
}

func TestGetImageData(t *testing.T) {
	term := New(WithSize(10, 20))

	imgData := []byte{
		255, 0, 0, 255,
		0, 255, 0, 255,
		0, 0, 255, 255,
		255, 255, 0, 255,
	}

	imgID := term.images.Store(2, 2, imgData)

	cases := []struct {
		name   string
		id     uint32
		exists bool
	}{
		{"found", imgID, true},
		{"not found", 999, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := term.GetImageData(c.id)
			if !c.exists {
				if result != nil {
					t.Errorf("Expected nil for non-existent image, got %v", result)
				}
				return
			}
			checkImageData(t, result, c.id, imgData)
		})
	}
}

func checkImageData(t *testing.T, result *ImageSnapshot, wantID uint32, imgData []byte) {
	t.Helper()
	if result == nil {
		t.Fatal("Expected image data, got nil")
	}
	if result.ID != wantID || result.Width != 2 || result.Height != 2 || result.Format != "rgba" {
		t.Errorf("Image metadata mismatch: %+v", result)
	}
	decoded, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	if len(decoded) != len(imgData) {
		t.Errorf("Decoded data length = %d, want %d", len(decoded), len(imgData))
	}
	for i, b := range decoded {
		if b != imgData[i] {
			t.Errorf("Decoded data[%d] = %d, want %d", i, b, imgData[i])
		}
	}
}
