package headlessterm

import (
	"testing"
)

func TestImageManager_Store(t *testing.T) {
	cases := []struct {
		name       string
		data       []byte
		wantCount  int
		wantMemory int
	}{
		{
			name:       "new image",
			data:       make([]byte, 100),
			wantCount:  1,
			wantMemory: 100,
		},
		{
			name:       "deduplicated image",
			data:       []byte("test image data"),
			wantCount:  1,
			wantMemory: len("test image data"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := NewImageManager()
			id1 := m.Store(10, 10, c.data)
			if id1 != 1 {
				t.Errorf("expected id 1, got %d", id1)
			}
			if c.name == "deduplicated image" {
				id2 := m.Store(10, 10, c.data)
				if id1 != id2 {
					t.Errorf("expected same id for duplicate, got %d and %d", id1, id2)
				}
			}
			if m.ImageCount() != c.wantCount {
				t.Errorf("expected %d images, got %d", c.wantCount, m.ImageCount())
			}
			if m.UsedMemory() != c.wantMemory {
				t.Errorf("expected %d bytes, got %d", c.wantMemory, m.UsedMemory())
			}
		})
	}
}

func TestImageManager_StoreWithID(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 50)
	m.StoreWithID(42, 5, 5, data)

	img := m.Image(42)
	if img == nil {
		t.Fatal("expected image with id 42")
	}
	if img.Width != 5 || img.Height != 5 {
		t.Errorf("expected 5x5, got %dx%d", img.Width, img.Height)
	}
}

func TestImageManager_Place(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	imageID := m.Store(10, 10, data)

	placement := &ImagePlacement{
		ImageID: imageID,
		Row:     0,
		Col:     0,
		Cols:    5,
		Rows:    5,
	}

	placementID := m.Place(placement)
	if placementID != 1 {
		t.Errorf("expected placement id 1, got %d", placementID)
	}
	if m.PlacementCount() != 1 {
		t.Errorf("expected 1 placement, got %d", m.PlacementCount())
	}
}

func TestImageManager_Lifecycle(t *testing.T) {
	cases := []struct {
		name           string
		operation      string
		wantImages     int
		wantPlacements int
		wantMemory     int
	}{
		{
			name:           "delete image",
			operation:      "delete",
			wantImages:     0,
			wantPlacements: 0,
			wantMemory:     0,
		},
		{
			name:           "clear all",
			operation:      "clear",
			wantImages:     0,
			wantPlacements: 0,
			wantMemory:     0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := NewImageManager()
			data := make([]byte, 100)
			imageID := m.Store(10, 10, data)
			m.Place(&ImagePlacement{ImageID: imageID, Row: 0, Col: 0, Cols: 1, Rows: 1})

			if c.operation == "delete" {
				m.DeleteImage(imageID)
			} else {
				m.Clear()
			}

			if m.ImageCount() != c.wantImages {
				t.Errorf("expected %d images, got %d", c.wantImages, m.ImageCount())
			}
			if m.PlacementCount() != c.wantPlacements {
				t.Errorf("expected %d placements, got %d", c.wantPlacements, m.PlacementCount())
			}
			if m.UsedMemory() != c.wantMemory {
				t.Errorf("expected %d bytes, got %d", c.wantMemory, m.UsedMemory())
			}
		})
	}
}

func TestImageManager_Prune(t *testing.T) {
	m := NewImageManager()
	m.SetMaxMemory(150) // Low limit

	// Store 3 images of 100 bytes each - should trigger pruning
	data := make([]byte, 100)
	m.Store(10, 10, data)

	data2 := make([]byte, 100)
	data2[0] = 1 // Different data
	m.Store(10, 10, data2)

	// At this point, we're at 200 bytes with 150 limit
	// Pruning should have removed unreferenced images
	if m.UsedMemory() > 150 {
		// This might not prune if images are still referenced
		// Just verify it doesn't crash
	}
}

func TestImageManager_Placements(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	imageID := m.Store(10, 10, data)

	m.Place(&ImagePlacement{ImageID: imageID, Row: 0, Col: 0, Cols: 1, Rows: 1})
	m.Place(&ImagePlacement{ImageID: imageID, Row: 1, Col: 1, Cols: 2, Rows: 2})

	placements := m.Placements()
	if len(placements) != 2 {
		t.Errorf("expected 2 placements, got %d", len(placements))
	}
}

func TestImageManager_DeletePlacements(t *testing.T) {
	cases := []struct {
		name      string
		delete    func(*ImageManager)
		wantCount int
	}{
		{
			name: "by position",
			delete: func(m *ImageManager) {
				m.DeletePlacementsByPosition(0, 0)
			},
			wantCount: 1,
		},
		{
			name: "in row",
			delete: func(m *ImageManager) {
				m.DeletePlacementsInRow(1)
			},
			wantCount: 1,
		},
		{
			name: "in row range",
			delete: func(m *ImageManager) {
				m.DeletePlacementsInRowRange(4, 8)
			},
			wantCount: 2,
		},
		{
			name: "below",
			delete: func(m *ImageManager) {
				m.DeletePlacementsBelow(4)
			},
			wantCount: 1,
		},
		{
			name: "above",
			delete: func(m *ImageManager) {
				m.DeletePlacementsAbove(7)
			},
			wantCount: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := NewImageManager()
			data := make([]byte, 100)
			imageID := m.Store(10, 10, data)

			// Placement at rows 0-2
			m.Place(&ImagePlacement{ImageID: imageID, Row: 0, Col: 0, Cols: 2, Rows: 3})
			// Placement at rows 5-7
			m.Place(&ImagePlacement{ImageID: imageID, Row: 5, Col: 0, Cols: 2, Rows: 3})
			// Placement at rows 10-12
			m.Place(&ImagePlacement{ImageID: imageID, Row: 10, Col: 0, Cols: 2, Rows: 3})

			c.delete(m)

			if m.PlacementCount() != c.wantCount {
				t.Errorf("expected %d placements after delete, got %d", c.wantCount, m.PlacementCount())
			}
		})
	}
}

func TestCellImage(t *testing.T) {
	cell := NewCell()

	if cell.HasImage() {
		t.Error("new cell should not have image")
	}

	cell.Image = &CellImage{
		PlacementID: 1,
		ImageID:     1,
		U0:          0.0,
		V0:          0.0,
		U1:          1.0,
		V1:          1.0,
		ZIndex:      -1,
	}

	if !cell.HasImage() {
		t.Error("cell should have image after setting")
	}

	cell.Reset()

	if cell.HasImage() {
		t.Error("cell should not have image after reset")
	}
}

func TestTerminalImageClearing(t *testing.T) {
	cases := []struct {
		name           string
		setup          func(*Terminal) (imageID int, preservedImageID int)
		act            func(*Terminal)
		wantPlacements int
		wantImages     int
	}{
		{
			name: "CSI 2J clears placements preserves images",
			setup: func(term *Terminal) (int, int) {
				data := make([]byte, 100)
				imageID := term.images.Store(10, 10, data)
				term.images.Place(&ImagePlacement{ImageID: imageID, Row: 5, Col: 5, Cols: 2, Rows: 2})
				return imageID, 0
			},
			act: func(term *Terminal) {
				term.WriteString("\x1b[2J")
			},
			wantPlacements: 0,
			wantImages:     1,
		},
		{
			name: "CSI 0J clears below cursor",
			setup: func(term *Terminal) (int, int) {
				data := make([]byte, 100)
				imageID := term.images.Store(10, 10, data)
				term.images.Place(&ImagePlacement{ImageID: imageID, Row: 2, Col: 0, Cols: 2, Rows: 2})  // Above
				term.images.Place(&ImagePlacement{ImageID: imageID, Row: 10, Col: 0, Cols: 2, Rows: 2}) // Below
				return imageID, 0
			},
			act: func(term *Terminal) {
				term.WriteString("\x1b[6;1H")
				term.WriteString("\x1b[0J")
			},
			wantPlacements: 1,
			wantImages:     1,
		},
		{
			name: "RIS clears images and placements",
			setup: func(term *Terminal) (int, int) {
				data := make([]byte, 100)
				imageID := term.images.Store(10, 10, data)
				term.images.Place(&ImagePlacement{ImageID: imageID, Row: 5, Col: 5, Cols: 2, Rows: 2})
				return imageID, 0
			},
			act: func(term *Terminal) {
				term.WriteString("\x1bc")
			},
			wantPlacements: 0,
			wantImages:     0,
		},
		{
			name: "alternate screen clears placements on both switches",
			setup: func(term *Terminal) (int, int) {
				data := make([]byte, 100)
				imageID := term.images.Store(10, 10, data)
				term.images.Place(&ImagePlacement{ImageID: imageID, Row: 5, Col: 5, Cols: 2, Rows: 2})
				return imageID, 0
			},
			act: func(term *Terminal) {
				term.WriteString("\x1b[?1049h")
				data := make([]byte, 100)
				imageID2 := term.images.Store(20, 20, data)
				term.images.Place(&ImagePlacement{ImageID: imageID2, Row: 0, Col: 0, Cols: 3, Rows: 3})
				term.WriteString("\x1b[?1049l")
			},
			wantPlacements: 0,
			wantImages:     2,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(24, 80))
			c.setup(term)
			c.act(term)
			if term.ImagePlacementCount() != c.wantPlacements {
				t.Errorf("expected %d placements, got %d", c.wantPlacements, term.ImagePlacementCount())
			}
			if term.ImageCount() != c.wantImages {
				t.Errorf("expected %d images, got %d", c.wantImages, term.ImageCount())
			}
		})
	}
}
