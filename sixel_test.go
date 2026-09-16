package headlessterm

import (
	"testing"
)

func TestParseSixel(t *testing.T) {
	cases := []struct {
		name            string
		params          []int64
		data            string
		wantWidth       uint32
		wantHeight      uint32
		wantTransparent bool
		checkPixel      bool
		wantR           byte
		wantG           byte
		wantB           byte
	}{
		{
			name:       "simple pixel",
			data:       "~",
			wantWidth:  1,
			wantHeight: 6,
		},
		{
			name:       "multiple columns",
			data:       "~~~",
			wantWidth:  3,
			wantHeight: 6,
		},
		{
			name:       "newline",
			data:       "~-~",
			wantWidth:  1,
			wantHeight: 12,
		},
		{
			name:       "carriage return",
			data:       "~$~",
			wantWidth:  1,
			wantHeight: 6,
		},
		{
			name:       "repeat",
			data:       "!5~",
			wantWidth:  5,
			wantHeight: 6,
		},
		{
			name:       "RGB color",
			data:       "#1;2;100;0;0#1~",
			wantWidth:  1,
			wantHeight: 6,
			checkPixel: true,
			wantR:      255,
			wantG:      0,
			wantB:      0,
		},
		{
			name:       "HLS color",
			data:       "#2;1;120;50;100#2~",
			wantWidth:  1,
			wantHeight: 6,
		},
		{
			name:            "transparent",
			params:          []int64{0, 1, 0},
			data:            "~",
			wantWidth:       1,
			wantHeight:      6,
			wantTransparent: true,
		},
		{
			name:       "empty",
			data:       "",
			wantWidth:  0,
			wantHeight: 0,
		},
		{
			name:       "complex image",
			data:       "#0;2;0;0;0#1;2;100;0;0#0!10~-#1!10~",
			wantWidth:  10,
			wantHeight: 12,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			img, err := ParseSixel(c.params, []byte(c.data))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if img.Width != c.wantWidth || img.Height != c.wantHeight {
				t.Errorf("dimensions = %dx%d, want %dx%d", img.Width, img.Height, c.wantWidth, c.wantHeight)
			}
			if img.Transparent != c.wantTransparent {
				t.Errorf("Transparent = %v, want %v", img.Transparent, c.wantTransparent)
			}
			if c.checkPixel {
				if len(img.Data) < 4 {
					t.Fatal("expected pixel data")
				}
				if img.Data[0] != c.wantR || img.Data[1] != c.wantG || img.Data[2] != c.wantB {
					t.Errorf("pixel = (%d,%d,%d), want (%d,%d,%d)", img.Data[0], img.Data[1], img.Data[2], c.wantR, c.wantG, c.wantB)
				}
			}
		})
	}
}

func TestSixelEndToEnd(t *testing.T) {
	cases := []struct {
		name           string
		setup          func(*Terminal)
		sixel          string
		wantImages     int
		wantPlacements int
		checkCells     bool
		cellRows       []int
		cellCols       []int
		wantCursorRow  int
	}{
		{
			name:           "display",
			setup:          func(term *Terminal) {},
			sixel:          "\x1bP0;0;0q#0;2;100;0;0#0!10~-!10~\x1b\\",
			wantImages:     1,
			wantPlacements: 1,
		},
		{
			name: "cell assignment",
			setup: func(term *Terminal) {
				term.WriteString("\x1b[3;6H")
			},
			sixel:          "\x1bP0;0;0q#0;2;100;0;0#0!20~-!20~\x1b\\",
			wantImages:     1,
			wantPlacements: 1,
			checkCells:     true,
			cellRows:       []int{2, 2, 3, 3},
			cellCols:       []int{5, 6, 5, 6},
		},
		{
			name: "cursor movement",
			setup: func(term *Terminal) {
				term.WriteString("\x1b[1;1H")
			},
			sixel:          "\x1bP0;0;0q!10~-!10~\x1b\\",
			wantImages:     1,
			wantPlacements: 1,
			wantCursorRow:  2,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(24, 80))
			term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})
			c.setup(term)
			term.WriteString(c.sixel)

			if term.ImageCount() != c.wantImages {
				t.Errorf("expected %d images, got %d", c.wantImages, term.ImageCount())
			}
			if term.ImagePlacementCount() != c.wantPlacements {
				t.Errorf("expected %d placements, got %d", c.wantPlacements, term.ImagePlacementCount())
			}
			if c.checkCells {
				for i := 0; i < len(c.cellRows); i++ {
					row, col := c.cellRows[i], c.cellCols[i]
					cell := term.Cell(row, col)
					if cell == nil {
						t.Fatalf("cell at %d,%d is nil", row, col)
					}
					if !cell.HasImage() {
						t.Errorf("expected cell at %d,%d to have image", row, col)
					}
					if cell.Char != ImagePlaceholderChar {
						t.Errorf("expected placeholder char at %d,%d, got %U", row, col, cell.Char)
					}
				}
			}
			if c.wantCursorRow > 0 {
				row, _ := term.CursorPos()
				if row != c.wantCursorRow {
					t.Errorf("cursor row = %d, want %d", row, c.wantCursorRow)
				}
			}
		})
	}
}

func TestSixelScrollingAtBottom(t *testing.T) {
	term := New(WithSize(10, 80))
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	term.WriteString("Line 1\r\n")
	term.WriteString("Line 2\r\n")
	term.WriteString("Line 3\r\n")

	term.WriteString("\x1b[10;1H")
	curRow, _ := term.CursorPos()
	if curRow != 9 {
		t.Fatalf("expected cursor at row 9, got %d", curRow)
	}

	sixel := "\x1bP0;0;0q!10~-!10~-!10~\x1b\\"
	term.WriteString(sixel)

	placements := term.ImagePlacements()
	if len(placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(placements))
	}
	p := placements[0]
	if p.Row+p.Rows > 10 {
		t.Errorf("placement extends beyond screen: row=%d, rows=%d", p.Row, p.Rows)
	}

	// Verify the screen scrolled: "Line 1" should no longer be at the top.
	if term.LineContent(0) == "Line 1" {
		t.Error("expected screen to scroll, but Line 1 is still at the top")
	}

	// The image cells should be visible somewhere on screen.
	found := false
	for row := 0; row < 10; row++ {
		for col := 0; col < 80; col++ {
			if cell := term.Cell(row, col); cell != nil && cell.HasImage() {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected image cells to be visible on screen")
	}
}
