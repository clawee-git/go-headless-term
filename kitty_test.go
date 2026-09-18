package headlessterm

import (
	"encoding/base64"
	"testing"
)

var parseKittyGraphicsCases = []struct {
	name  string
	data  string
	check func(t *testing.T, cmd *KittyCommand)
}{
	{
		name: "basic transmit and display",
		data: "Ga=T,f=32,s=2,v=2;AAAAAAAAAAAAAAAAAAAAAAA=",
		check: func(t *testing.T, cmd *KittyCommand) {
			if cmd.Action != KittyActionTransmitDisplay {
				t.Errorf("expected action T, got %c", cmd.Action)
			}
			if cmd.Format != KittyFormatRGBA {
				t.Errorf("expected format 32, got %d", cmd.Format)
			}
			if cmd.Width != 2 {
				t.Errorf("expected width 2, got %d", cmd.Width)
			}
			if cmd.Height != 2 {
				t.Errorf("expected height 2, got %d", cmd.Height)
			}
		},
	},
	{
		name: "query",
		data: "Ga=q,i=1;",
		check: func(t *testing.T, cmd *KittyCommand) {
			if cmd.Action != KittyActionQuery {
				t.Errorf("expected action q, got %c", cmd.Action)
			}
			if cmd.ImageID != 1 {
				t.Errorf("expected image ID 1, got %d", cmd.ImageID)
			}
		},
	},
	{
		name: "delete",
		data: "Ga=d,d=a;",
		check: func(t *testing.T, cmd *KittyCommand) {
			if cmd.Action != KittyActionDelete {
				t.Errorf("expected action d, got %c", cmd.Action)
			}
			if cmd.Delete != KittyDeleteAll {
				t.Errorf("expected delete all, got %c", cmd.Delete)
			}
		},
	},
	{
		name: "chunked",
		data: "Ga=T,m=1;AAAA",
		check: func(t *testing.T, cmd *KittyCommand) {
			if !cmd.More {
				t.Error("expected more=true")
			}
		},
	},
	{
		name: "with z-index",
		data: "Ga=p,i=1,z=-1;",
		check: func(t *testing.T, cmd *KittyCommand) {
			if cmd.ZIndex != -1 {
				t.Errorf("expected z-index -1, got %d", cmd.ZIndex)
			}
		},
	},
	{
		name: "placement",
		data: "Ga=p,i=1,c=10,r=5,X=2,Y=3;",
		check: func(t *testing.T, cmd *KittyCommand) {
			if cmd.Cols != 10 {
				t.Errorf("expected cols 10, got %d", cmd.Cols)
			}
			if cmd.Rows != 5 {
				t.Errorf("expected rows 5, got %d", cmd.Rows)
			}
			if cmd.CellOffsetX != 2 {
				t.Errorf("expected offsetX 2, got %d", cmd.CellOffsetX)
			}
			if cmd.CellOffsetY != 3 {
				t.Errorf("expected offsetY 3, got %d", cmd.CellOffsetY)
			}
		},
	},
	{
		name: "do not move cursor",
		data: "Ga=T,C=1;",
		check: func(t *testing.T, cmd *KittyCommand) {
			if !cmd.DoNotMoveCursor {
				t.Error("expected DoNotMoveCursor=true")
			}
		},
	},
}

func TestParseKittyGraphics(t *testing.T) {
	for _, c := range parseKittyGraphicsCases {
		t.Run(c.name, func(t *testing.T) {
			cmd, err := ParseKittyGraphics([]byte(c.data))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			c.check(t, cmd)
		})
	}
}

var kittyDecodeImageDataCases = []struct {
	name      string
	format    KittyFormat
	pixelByte int // bytes per pixel in the payload
	fill      byte
	wantLen   int
	wantAlpha byte // checked when > 0
}{
	{name: "rgba", format: KittyFormatRGBA, pixelByte: 4, fill: 255, wantLen: 16},
	{name: "rgb", format: KittyFormatRGB, pixelByte: 3, fill: 128, wantLen: 16, wantAlpha: 255},
}

func TestKittyCommand_DecodeImageData(t *testing.T) {
	for _, c := range kittyDecodeImageDataCases {
		t.Run(c.name, func(t *testing.T) {
			// 2x2 image in the case's format
			payload := make([]byte, 2*2*c.pixelByte)
			for i := range payload {
				payload[i] = c.fill
			}

			cmd := &KittyCommand{
				Format:  c.format,
				Width:   2,
				Height:  2,
				Payload: payload,
			}

			data, w, h, err := cmd.DecodeImageData()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w != 2 || h != 2 {
				t.Errorf("expected 2x2, got %dx%d", w, h)
			}
			if len(data) != c.wantLen {
				t.Errorf("expected %d bytes, got %d", c.wantLen, len(data))
			}
			if c.wantAlpha > 0 && data[3] != c.wantAlpha {
				t.Errorf("expected alpha %d, got %d", c.wantAlpha, data[3])
			}
		})
	}
}

func TestFormatKittyResponse(t *testing.T) {
	resp := FormatKittyResponse(42, "", false)
	expected := "\x1b_Gi=42;OK\x1b\\"
	if resp != expected {
		t.Errorf("expected %q, got %q", expected, resp)
	}

	respErr := FormatKittyResponse(0, "ENOENT", true)
	expectedErr := "\x1b_G;ENOENT\x1b\\"
	if respErr != expectedErr {
		t.Errorf("expected %q, got %q", expectedErr, respErr)
	}
}

func TestKittyImageEndToEnd(t *testing.T) {
	t.Run("display", kittyImageDisplayCase)
	t.Run("cell assignment", kittyImageCellAssignmentCase)
	t.Run("uv coordinates", kittyImageUVCoordinatesCase)
	t.Run("chunked transfer", kittyChunkedTransferCase)
	t.Run("image delete", kittyImageDeleteCase)
}

func kittyImageDisplayCase(t *testing.T) {
	term := New(WithSize(24, 80))

	// A 2x2 all-white RGBA image
	payload := kittyRGBAPayload(2, 2, 255)

	// Send Kitty graphics command via APC sequence
	// a=T (transmit and display), f=32 (RGBA), s=2 (width), v=2 (height)
	apc := "\x1b_Ga=T,f=32,s=2,v=2;" + payload + "\x1b\\"
	term.WriteString(apc)

	// Verify image was stored
	if term.ImageCount() != 1 {
		t.Errorf("expected 1 image, got %d", term.ImageCount())
	}

	// Verify placement was created
	if term.ImagePlacementCount() != 1 {
		t.Errorf("expected 1 placement, got %d", term.ImagePlacementCount())
	}
}

func kittyImageCellAssignmentCase(t *testing.T) {
	term := New(WithSize(24, 80))

	// Set cell size for calculations (20x20 pixels per cell)
	term.SetSizeProvider(&testSizeProvider{cellW: 20, cellH: 20})

	// A 40x40 RGBA image covers 2x2 cells at 20x20 pixels per cell
	payload := kittyRGBAPayload(40, 40, 128)

	// Position cursor at row 5, col 10
	term.WriteString("\x1b[6;11H") // 1-based positioning

	// Transmit and display
	apc := "\x1b_Ga=T,f=32,s=40,v=40;" + payload + "\x1b\\"
	term.WriteString(apc)

	// Check cells have image references
	for row := 5; row < 7; row++ {
		for col := 10; col < 12; col++ {
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
}

func kittyImageUVCoordinatesCase(t *testing.T) {
	term := New(WithSize(24, 80))

	// Set cell size (10x10 pixels per cell)
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// A 20x20 RGBA image covers 2x2 cells exactly at 10x10 pixels per cell
	payload := kittyRGBAPayload(20, 20, 0)

	// Position cursor at origin
	term.WriteString("\x1b[1;1H")

	// Transmit and display
	apc := "\x1b_Ga=T,f=32,s=20,v=20;" + payload + "\x1b\\"
	term.WriteString(apc)

	// Check UV coordinates for each cell
	testCases := []struct {
		row, col       int
		u0, v0, u1, v1 float32
	}{
		{0, 0, 0.0, 0.0, 0.5, 0.5}, // Top-left
		{0, 1, 0.5, 0.0, 1.0, 0.5}, // Top-right
		{1, 0, 0.0, 0.5, 0.5, 1.0}, // Bottom-left
		{1, 1, 0.5, 0.5, 1.0, 1.0}, // Bottom-right
	}

	for _, tc := range testCases {
		cell := term.Cell(tc.row, tc.col)
		if cell == nil || cell.Image == nil {
			t.Fatalf("cell at %d,%d has no image", tc.row, tc.col)
		}

		img := cell.Image
		if !floatClose(img.U0, tc.u0) || !floatClose(img.V0, tc.v0) ||
			!floatClose(img.U1, tc.u1) || !floatClose(img.V1, tc.v1) {
			t.Errorf("cell %d,%d: expected UV (%v,%v)-(%v,%v), got (%v,%v)-(%v,%v)",
				tc.row, tc.col, tc.u0, tc.v0, tc.u1, tc.v1,
				img.U0, img.V0, img.U1, img.V1)
		}
	}
}

func kittyChunkedTransferCase(t *testing.T) {
	term := New(WithSize(24, 80))
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Create a 20x20 RGBA image
	width, height := 20, 20
	rgba := make([]byte, width*height*4)
	for i := range rgba {
		rgba[i] = uint8(i % 256)
	}

	// Split into chunks
	chunk1 := rgba[:800]
	chunk2 := rgba[800:]
	payload1 := base64.StdEncoding.EncodeToString(chunk1)
	payload2 := base64.StdEncoding.EncodeToString(chunk2)

	// First chunk with m=1 (more coming)
	apc1 := "\x1b_Ga=T,f=32,s=20,v=20,m=1;" + payload1 + "\x1b\\"
	term.WriteString(apc1)

	// No image yet (incomplete)
	if term.ImageCount() != 0 {
		t.Errorf("expected 0 images during chunked transfer, got %d", term.ImageCount())
	}

	// Final chunk with m=0 (or no m)
	apc2 := "\x1b_Gm=0;" + payload2 + "\x1b\\"
	term.WriteString(apc2)

	// Now image should be complete
	if term.ImageCount() != 1 {
		t.Errorf("expected 1 image after chunked transfer, got %d", term.ImageCount())
	}
}

func kittyImageDeleteCase(t *testing.T) {
	term := New(WithSize(24, 80))
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Create and display an image
	payload := kittyRGBAPayload(10, 10, 0)
	apc := "\x1b_Ga=T,f=32,s=10,v=10,i=42;" + payload + "\x1b\\"
	term.WriteString(apc)

	if term.ImageCount() != 1 {
		t.Fatalf("expected 1 image, got %d", term.ImageCount())
	}

	// Delete all visible placements
	term.WriteString("\x1b_Ga=d,d=a;\x1b\\")

	// Placements should be removed
	if term.ImagePlacementCount() != 0 {
		t.Errorf("expected 0 placements after delete, got %d", term.ImagePlacementCount())
	}

	// Image data should still exist (d=a only deletes placements)
	if term.ImageCount() != 1 {
		t.Errorf("expected 1 image after placement delete, got %d", term.ImageCount())
	}

	// Delete image data too
	term.WriteString("\x1b_Ga=d,d=I,i=42;\x1b\\")

	if term.ImageCount() != 0 {
		t.Errorf("expected 0 images after data delete, got %d", term.ImageCount())
	}
}

// testSizeProvider is a test implementation of SizeProvider
type testSizeProvider struct {
	cellW, cellH int
}

func (p *testSizeProvider) CellSizePixels() (width, height int) {
	return p.cellW, p.cellH
}

func (p *testSizeProvider) WindowSizePixels() (width, height int) {
	return 800, 600
}

// floatClose checks if two floats are approximately equal
func floatClose(a, b float32) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.01
}

// kittyRGBAPayload is the base64 transmission payload of a w x h RGBA image
// whose every byte is fill.
func kittyRGBAPayload(w, h int, fill byte) string {
	rgba := make([]byte, w*h*4)
	for i := range rgba {
		rgba[i] = fill
	}
	return base64.StdEncoding.EncodeToString(rgba)
}
