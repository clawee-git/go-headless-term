package headlessterm

import (
	"strings"
	"testing"
)

func TestTerminalResize(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("Hello")
	term.Resize(10, 40)

	if term.Rows() != 10 || term.Cols() != 40 {
		t.Errorf("expected size 10x40, got %dx%d", term.Rows(), term.Cols())
	}

	// Content should be preserved
	if term.LineContent(0) != "Hello" {
		t.Errorf("expected content preserved after resize, got '%s'", term.LineContent(0))
	}
}

func TestTerminalAutoResizeY(t *testing.T) {
	term := New(WithSize(3, 80), WithAutoResize())

	if !term.AutoResize() {
		t.Error("expected AutoResize to be enabled")
	}

	// Write more lines than terminal height (use \r\n for proper line breaks)
	term.WriteString("Line1\r\n")
	term.WriteString("Line2\r\n")
	term.WriteString("Line3\r\n")
	term.WriteString("Line4\r\n")
	term.WriteString("Line5\r\n")

	// Terminal should have grown
	if term.Rows() < 5 {
		t.Errorf("expected at least 5 rows, got %d", term.Rows())
	}

	// All content should be in the buffer (no scrolling)
	if term.LineContent(0) != "Line1" {
		t.Errorf("expected 'Line1', got '%s'", term.LineContent(0))
	}
	if term.LineContent(4) != "Line5" {
		t.Errorf("expected 'Line5', got '%s'", term.LineContent(4))
	}
}

func TestTerminalAutoResizeX(t *testing.T) {
	term := New(WithSize(3, 10), WithAutoResize())

	// Write a line longer than terminal width
	term.WriteString("This is a very long line that exceeds the terminal width")

	// Terminal should have grown horizontally
	if term.Cols() <= 10 {
		t.Errorf("expected cols > 10, got %d", term.Cols())
	}

	// Content should be on single line (no wrap)
	content := term.LineContent(0)
	if content != "This is a very long line that exceeds the terminal width" {
		t.Errorf("expected full line, got '%s'", content)
	}

	// Cursor should still be on line 0
	row, _ := term.CursorPos()
	if row != 0 {
		t.Errorf("expected cursor on row 0, got %d", row)
	}
}

func TestTerminalAutoResizeNoScrollback(t *testing.T) {
	term, storage := newScrollbackTerm(3, WithAutoResize())

	// Write many lines (use \r\n for proper line breaks)
	for i := 0; i < 10; i++ {
		term.WriteString("Line\r\n")
	}

	// With AutoResize, nothing should go to scrollback
	if storage.pushCount > 0 {
		t.Errorf("expected no scrollback pushes with AutoResize, got %d", storage.pushCount)
	}
}

// TestResizeInvalidDimensions tests that Resize ignores invalid dimensions
func TestTerminalResizeBounds(t *testing.T) {
	t.Run("invalid dimensions", resizeInvalidDimensionsCase)
	t.Run("cursor clamped after resize", resizeCursorBoundsCase)
	t.Run("bounds after grow cols", cursorBoundsAfterGrowColsCase)
	t.Run("bounds after wrap", cursorBoundsAfterWrapCase)
	t.Run("invalid cursor position", inputWithInvalidCursorPositionCase)
}

func resizeInvalidDimensionsCase(t *testing.T) {
	term := New(WithSize(24, 80))

	originalRows := term.Rows()
	originalCols := term.Cols()

	// Test with zero dimensions
	term.Resize(0, 0)
	if term.Rows() != originalRows || term.Cols() != originalCols {
		t.Errorf("Resize(0, 0) should be ignored, got %dx%d", term.Rows(), term.Cols())
	}

	// Test with negative dimensions
	term.Resize(-10, -20)
	if term.Rows() != originalRows || term.Cols() != originalCols {
		t.Errorf("Resize(-10, -20) should be ignored, got %dx%d", term.Rows(), term.Cols())
	}

	// Test with zero rows
	term.Resize(0, 100)
	if term.Rows() != originalRows || term.Cols() != originalCols {
		t.Errorf("Resize(0, 100) should be ignored, got %dx%d", term.Rows(), term.Cols())
	}

	// Test with zero cols
	term.Resize(50, 0)
	if term.Rows() != originalRows || term.Cols() != originalCols {
		t.Errorf("Resize(50, 0) should be ignored, got %dx%d", term.Rows(), term.Cols())
	}

	// Test with valid dimensions
	term.Resize(30, 100)
	if term.Rows() != 30 || term.Cols() != 100 {
		t.Errorf("Resize(30, 100) should work, got %dx%d", term.Rows(), term.Cols())
	}
}

// resizeCursorBoundsCase tests that cursor is properly clamped after resize
func resizeCursorBoundsCase(t *testing.T) {
	term := New(WithSize(24, 80))

	// Move cursor to end
	term.WriteString(strings.Repeat("A", 80))
	term.WriteString("\r\n")
	term.WriteString(strings.Repeat("B", 80))

	// Resize to smaller dimensions
	term.Resize(10, 40)

	row, col := term.CursorPos()
	if row < 0 || row >= 10 {
		t.Errorf("cursor row out of bounds after resize: %d (expected 0-9)", row)
	}
	if col < 0 || col >= 40 {
		t.Errorf("cursor col out of bounds after resize: %d (expected 0-39)", col)
	}
}

// cursorBoundsAfterGrowColsCase tests that cursor stays within bounds after auto-resize
func cursorBoundsAfterGrowColsCase(t *testing.T) {
	term := New(WithSize(5, 10), WithAutoResize())

	// Write a wide character at the end of line (should trigger GrowCols)
	term.WriteString(strings.Repeat("A", 9)) // Fill 9 columns
	term.WriteString("中")                    // Wide character (2 columns) at position 9

	row, col := term.CursorPos()
	if row < 0 || row >= term.Rows() {
		t.Errorf("cursor row out of bounds after GrowCols: %d (rows: %d)", row, term.Rows())
	}
	if col < 0 || col > term.Cols() {
		t.Errorf("cursor col out of bounds after GrowCols: %d (cols: %d)", col, term.Cols())
	}

	// Verify the character was written
	content := term.LineContent(0)
	if len(content) < 10 {
		t.Errorf("expected line to grow, got length %d", len(content))
	}
}

// cursorBoundsAfterWrapCase tests that cursor row is validated after line wrap
func cursorBoundsAfterWrapCase(t *testing.T) {
	term := New(WithSize(5, 10))

	// Fill terminal with text to trigger wrapping
	for i := 0; i < 10; i++ {
		term.WriteString("123456789") // 9 chars, will wrap on next char
		term.WriteString("A")         // Triggers wrap
	}

	row, col := term.CursorPos()
	if row < 0 || row >= term.Rows() {
		t.Errorf("cursor row out of bounds after wrap: %d (rows: %d)", row, term.Rows())
	}
	if col < 0 || col > term.Cols() {
		t.Errorf("cursor col out of bounds after wrap: %d (cols: %d)", col, term.Cols())
	}
}

// inputWithInvalidCursorPositionCase tests that input handles invalid cursor positions gracefully
func inputWithInvalidCursorPositionCase(t *testing.T) {
	term := New(WithSize(5, 10))

	// Manually set cursor to invalid position (would require reflection, but we test indirectly)
	// by writing characters that would cause cursor to go out of bounds

	// Write to fill terminal
	for i := 0; i < 100; i++ {
		term.WriteString("A")
	}

	// Cursor should still be within bounds (allow col == cols for edge case)
	row, col := term.CursorPos()
	if row < 0 || row >= term.Rows() {
		t.Errorf("cursor row out of bounds: %d (rows: %d)", row, term.Rows())
	}
	if col < 0 || col > term.Cols() {
		t.Errorf("cursor col out of bounds: %d (cols: %d)", col, term.Cols())
	}

	// Verify we can still write without panic
	term.WriteString("X")
	row2, col2 := term.CursorPos()
	if row2 < 0 || row2 >= term.Rows() || col2 < 0 || col2 > term.Cols() {
		t.Errorf("cursor out of bounds after write: (%d, %d)", row2, col2)
	}
}

func TestTerminalResizeScrollback(t *testing.T) {
	t.Run("grow pulls from scrollback", resizeGrowPullsFromScrollbackCase)
	t.Run("shrink with cursor in bounds", resizeShrinkCursorInBoundsCase)
	t.Run("shrink with cursor out of bounds", resizeShrinkCursorOutOfBoundsCase)
	t.Run("shrink scrollback content", resizeShrinkScrollbackContentCase)
	t.Run("grow no scrollback unchanged", resizeGrowNoScrollbackUnchangedCase)
	t.Run("cursor position after shrink", resizeCursorPositionAfterShrinkCase)
}

// resizeShrinkCursorInBoundsCase tests resize when cursor stays within new bounds
func resizeShrinkCursorInBoundsCase(t *testing.T) {
	term, storage := newScrollbackTerm(10)

	// Write content on first few lines
	term.WriteString("Line0\r\n")
	term.WriteString("Line1\r\n")
	term.WriteString("Line2")

	// Cursor should be on row 2
	row, _ := term.CursorPos()
	if row != 2 {
		t.Errorf("expected cursor on row 2, got %d", row)
	}

	initialScrollbackLen := storage.Len()

	// Resize to 5 rows - cursor is still within bounds (row 2 < 5)
	term.Resize(5, 80)

	// No lines should be scrolled to scrollback
	if storage.Len() != initialScrollbackLen {
		t.Errorf("expected no new scrollback lines, got %d new lines", storage.Len()-initialScrollbackLen)
	}

	// Content should be preserved
	if term.LineContent(0) != "Line0" {
		t.Errorf("expected 'Line0', got '%s'", term.LineContent(0))
	}
	if term.LineContent(2) != "Line2" {
		t.Errorf("expected 'Line2', got '%s'", term.LineContent(2))
	}

	// Cursor should still be on row 2
	row, _ = term.CursorPos()
	if row != 2 {
		t.Errorf("expected cursor on row 2 after resize, got %d", row)
	}
}

// resizeShrinkCursorOutOfBoundsCase tests resize when cursor would be outside new bounds
func resizeShrinkCursorOutOfBoundsCase(t *testing.T) {
	term, storage := newScrollbackTerm(10)

	// Write content filling more lines
	writeNumberedLines(term, "Line", 9) // cursor ends on row 8

	row, _ := term.CursorPos()
	if row != 8 {
		t.Errorf("expected cursor on row 8, got %d", row)
	}

	initialScrollbackLen := storage.Len()

	// Resize to 5 rows - cursor at row 8 is outside bounds
	// Lines should be scrolled to preserve content near cursor
	term.Resize(5, 80)

	// Lines should have been pushed to scrollback
	newScrollbackLines := storage.Len() - initialScrollbackLen
	if newScrollbackLines == 0 {
		t.Error("expected lines to be pushed to scrollback when cursor is out of bounds")
	}

	// Cursor should be within new bounds
	row, _ = term.CursorPos()
	if row < 0 || row >= 5 {
		t.Errorf("cursor row out of bounds after resize: %d (expected 0-4)", row)
	}

	// Content near cursor should be preserved (Line8 should still be visible)
	found := false
	for i := 0; i < 5; i++ {
		if strings.Contains(term.LineContent(i), "Line8") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Line8 (near cursor) to be preserved after resize")
	}
}

// resizeShrinkScrollbackContentCase tests that scrolled lines go to scrollback correctly
func resizeShrinkScrollbackContentCase(t *testing.T) {
	term, storage := newScrollbackTerm(10)

	// Write content on all lines
	writeNumberedLines(term, "Line", 10)

	// Cursor on row 9
	row, _ := term.CursorPos()
	if row != 9 {
		t.Errorf("expected cursor on row 9, got %d", row)
	}

	// Resize to 5 rows
	term.Resize(5, 80)

	// Check scrollback contains the lines that were pushed
	if storage.Len() < 5 {
		t.Errorf("expected at least 5 lines in scrollback, got %d", storage.Len())
	}

	// First lines should be in scrollback
	foundLine0 := false
	for i := 0; i < storage.Len(); i++ {
		line := storage.Line(i)
		if line != nil && len(line) > 0 {
			content := ""
			for _, cell := range line {
				if cell.Char != 0 && cell.Char != ' ' {
					content += string(cell.Char)
				}
			}
			if strings.HasPrefix(content, "Line0") {
				foundLine0 = true
				break
			}
		}
	}
	if !foundLine0 {
		t.Error("expected Line0 to be in scrollback after resize")
	}
}

// resizeGrowPullsFromScrollbackCase tests that growing terminal pulls lines from scrollback
func resizeGrowPullsFromScrollbackCase(t *testing.T) {
	term, storage := newScrollbackTerm(10)

	// Write content that fills the terminal
	writeNumberedLines(term, "Line", 10)

	// Shrink terminal - this pushes lines to scrollback
	term.Resize(5, 80)

	scrollbackAfterShrink := storage.Len()
	if scrollbackAfterShrink == 0 {
		t.Fatal("expected lines in scrollback after shrink")
	}

	// Grow terminal back - should pull lines from scrollback
	term.Resize(10, 80)

	// Scrollback should have been consumed
	if storage.Len() >= scrollbackAfterShrink {
		t.Errorf("expected scrollback to be consumed, was %d now %d", scrollbackAfterShrink, storage.Len())
	}

	// Content should be restored at top of screen
	foundLine0 := false
	for i := 0; i < 10; i++ {
		if strings.Contains(term.LineContent(i), "Line0") {
			foundLine0 = true
			break
		}
	}
	if !foundLine0 {
		t.Error("expected Line0 to be restored from scrollback")
	}
}

// resizeGrowNoScrollbackUnchangedCase tests that growing terminal without scrollback doesn't change content
func resizeGrowNoScrollbackUnchangedCase(t *testing.T) {
	term, storage := newScrollbackTerm(5)

	// Write some content (not enough to need scrollback)
	term.WriteString("Line0\r\n")
	term.WriteString("Line1\r\n")
	term.WriteString("Line2")

	row, _ := term.CursorPos()
	initialRow := row
	initialScrollbackLen := storage.Len()

	// Grow terminal
	term.Resize(10, 80)

	// Scrollback should remain empty
	if storage.Len() != initialScrollbackLen {
		t.Errorf("expected scrollback unchanged, was %d now %d", initialScrollbackLen, storage.Len())
	}

	// Cursor should stay at same position (no lines pulled)
	row, _ = term.CursorPos()
	if row != initialRow {
		t.Errorf("expected cursor to stay at row %d, got %d", initialRow, row)
	}

	// Content should be preserved
	if term.LineContent(0) != "Line0" {
		t.Errorf("expected 'Line0', got '%s'", term.LineContent(0))
	}
}

// TestResizeAlternateScreenNoScrollback tests that alternate screen doesn't use scrollback on resize
func TestResizeAlternateScreenNoScrollback(t *testing.T) {
	term, storage := newScrollbackTerm(10)

	// Switch to alternate screen
	term.WriteString("\x1b[?1049h")

	// Write content
	writeNumberedLines(term, "Alt", 9)

	initialScrollbackLen := storage.Len()

	// Resize - alternate screen should NOT push to scrollback
	term.Resize(5, 80)

	// Scrollback should not have new lines from alternate screen
	if storage.Len() != initialScrollbackLen {
		t.Errorf("alternate screen should not push to scrollback, was %d now %d",
			initialScrollbackLen, storage.Len())
	}
}

// resizeCursorPositionAfterShrinkCase tests cursor position is correctly adjusted
func resizeCursorPositionAfterShrinkCase(t *testing.T) {
	term, _ := newScrollbackTerm(20)

	// Move cursor to row 15
	for i := 0; i < 15; i++ {
		term.WriteString("Line\r\n")
	}
	term.WriteString("CursorLine")

	row, _ := term.CursorPos()
	if row != 15 {
		t.Errorf("expected cursor on row 15, got %d", row)
	}

	// Resize to 10 rows
	term.Resize(10, 80)

	// Cursor should be adjusted (15 - (20-10) = 5, but clamped to valid range)
	row, _ = term.CursorPos()
	if row < 0 || row >= 10 {
		t.Errorf("cursor out of bounds after resize: %d", row)
	}

	// The line with "CursorLine" should still be visible
	found := false
	for i := 0; i < 10; i++ {
		if strings.Contains(term.LineContent(i), "CursorLine") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected CursorLine to be visible after resize")
	}
}

// --- Row Coordinate Conversion Tests ---

func TestRowCoordinateConversion(t *testing.T) {
	t.Run("viewport to absolute", viewportRowToAbsoluteCase)
	t.Run("absolute to viewport", absoluteRowToViewportCase)
	t.Run("round trip", rowConversionRoundTripCase)
}

func viewportRowToAbsoluteCase(t *testing.T) {
	term, _ := newScrollbackTerm(5)

	// Without scrollback, viewport row equals absolute row
	if got := term.ViewportRowToAbsolute(0); got != 0 {
		t.Errorf("without scrollback: expected 0, got %d", got)
	}
	if got := term.ViewportRowToAbsolute(3); got != 3 {
		t.Errorf("without scrollback: expected 3, got %d", got)
	}

	// Create scrollback by writing more lines than terminal height
	for i := 0; i < 10; i++ {
		term.WriteString("Line\n")
	}

	scrollbackLen := term.ScrollbackLen()
	if scrollbackLen == 0 {
		t.Fatal("expected scrollback to exist")
	}

	// Viewport row 0 should now be at absolute row = scrollbackLen
	if got := term.ViewportRowToAbsolute(0); got != scrollbackLen {
		t.Errorf("with scrollback: expected %d, got %d", scrollbackLen, got)
	}

	// Viewport row 2 should be at absolute row = scrollbackLen + 2
	if got := term.ViewportRowToAbsolute(2); got != scrollbackLen+2 {
		t.Errorf("with scrollback: expected %d, got %d", scrollbackLen+2, got)
	}
}

func absoluteRowToViewportCase(t *testing.T) {
	term, _ := newScrollbackTerm(5)

	// Without scrollback, absolute row equals viewport row
	if got := term.AbsoluteRowToViewport(0); got != 0 {
		t.Errorf("without scrollback: expected 0, got %d", got)
	}
	if got := term.AbsoluteRowToViewport(3); got != 3 {
		t.Errorf("without scrollback: expected 3, got %d", got)
	}

	// Out of bounds should return -1
	if got := term.AbsoluteRowToViewport(5); got != -1 {
		t.Errorf("out of bounds: expected -1, got %d", got)
	}
	if got := term.AbsoluteRowToViewport(-1); got != -1 {
		t.Errorf("negative: expected -1, got %d", got)
	}

	// Create scrollback
	for i := 0; i < 10; i++ {
		term.WriteString("Line\n")
	}

	scrollbackLen := term.ScrollbackLen()

	// Rows in scrollback should return -1
	if got := term.AbsoluteRowToViewport(0); got != -1 {
		t.Errorf("scrollback row: expected -1, got %d", got)
	}
	if got := term.AbsoluteRowToViewport(scrollbackLen - 1); got != -1 {
		t.Errorf("last scrollback row: expected -1, got %d", got)
	}

	// First visible row
	if got := term.AbsoluteRowToViewport(scrollbackLen); got != 0 {
		t.Errorf("first visible: expected 0, got %d", got)
	}

	// Row in middle of viewport
	if got := term.AbsoluteRowToViewport(scrollbackLen + 2); got != 2 {
		t.Errorf("middle viewport: expected 2, got %d", got)
	}

	// Row beyond viewport
	if got := term.AbsoluteRowToViewport(scrollbackLen + 10); got != -1 {
		t.Errorf("beyond viewport: expected -1, got %d", got)
	}
}

func rowConversionRoundTripCase(t *testing.T) {
	term, _ := newScrollbackTerm(5)

	// Create some scrollback
	for i := 0; i < 10; i++ {
		term.WriteString("Line\n")
	}

	// Round trip: viewport -> absolute -> viewport should return original
	for viewportRow := 0; viewportRow < 5; viewportRow++ {
		absRow := term.ViewportRowToAbsolute(viewportRow)
		backToViewport := term.AbsoluteRowToViewport(absRow)
		if backToViewport != viewportRow {
			t.Errorf("round trip failed: viewport %d -> abs %d -> viewport %d",
				viewportRow, absRow, backToViewport)
		}
	}
}
