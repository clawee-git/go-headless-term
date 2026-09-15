package headlessterm

import (
	"image/color"
	"testing"
)

func TestColorsResolution(t *testing.T) {
	t.Run("DefaultPalette standard colors", func(t *testing.T) {
		cases := []struct {
			idx      int
			expected color.RGBA
		}{
			{0, color.RGBA{0, 0, 0, 255}},
			{1, color.RGBA{205, 49, 49, 255}},
			{2, color.RGBA{13, 188, 121, 255}},
			{3, color.RGBA{229, 229, 16, 255}},
			{4, color.RGBA{36, 114, 200, 255}},
			{5, color.RGBA{188, 63, 188, 255}},
			{6, color.RGBA{17, 168, 205, 255}},
			{7, color.RGBA{229, 229, 229, 255}},
			{8, color.RGBA{102, 102, 102, 255}},
			{15, color.RGBA{255, 255, 255, 255}},
		}
		for _, c := range cases {
			got := DefaultPalette[c.idx]
			if got != c.expected {
				t.Errorf("DefaultPalette[%d] = %+v, want %+v", c.idx, got, c.expected)
			}
		}
	})

	t.Run("DefaultPalette color cube", func(t *testing.T) {
		for r := 0; r < 6; r++ {
			for g := 0; g < 6; g++ {
				for b := 0; b < 6; b++ {
					idx := 16 + r*36 + g*6 + b
					got := DefaultPalette[idx]
					want := color.RGBA{uint8(r * 51), uint8(g * 51), uint8(b * 51), 255}
					if got != want {
						t.Errorf("DefaultPalette[%d] = %+v, want %+v", idx, got, want)
					}
				}
			}
		}
	})

	t.Run("DefaultPalette grayscale", func(t *testing.T) {
		for j := 0; j < 24; j++ {
			idx := 232 + j
			g := uint8(8 + j*10)
			got := DefaultPalette[idx]
			want := color.RGBA{g, g, g, 255}
			if got != want {
				t.Errorf("DefaultPalette[%d] = %+v, want %+v", idx, got, want)
			}
		}
	})

	t.Run("ResolveDefaultColor nil", func(t *testing.T) {
		if got := ResolveDefaultColor(nil, true); got != DefaultForeground {
			t.Errorf("ResolveDefaultColor(nil, true) = %+v, want %+v", got, DefaultForeground)
		}
		if got := ResolveDefaultColor(nil, false); got != DefaultBackground {
			t.Errorf("ResolveDefaultColor(nil, false) = %+v, want %+v", got, DefaultBackground)
		}
	})

	t.Run("ResolveDefaultColor RGBA passthrough", func(t *testing.T) {
		c := color.RGBA{1, 2, 3, 4}
		if got := ResolveDefaultColor(c, true); got != c {
			t.Errorf("ResolveDefaultColor(RGBA) = %+v, want %+v", got, c)
		}
	})

	t.Run("ResolveDefaultColor indexed", func(t *testing.T) {
		idx := &IndexedColor{Index: 3}
		if got := ResolveDefaultColor(idx, true); got != DefaultPalette[3] {
			t.Errorf("ResolveDefaultColor(indexed) = %+v, want %+v", got, DefaultPalette[3])
		}
	})

	t.Run("ResolveDefaultColor indexed out of range", func(t *testing.T) {
		idx := &IndexedColor{Index: 256}
		if got := ResolveDefaultColor(idx, true); got != DefaultForeground {
			t.Errorf("ResolveDefaultColor(256, true) = %+v, want %+v", got, DefaultForeground)
		}
		if got := ResolveDefaultColor(idx, false); got != DefaultBackground {
			t.Errorf("ResolveDefaultColor(256, false) = %+v, want %+v", got, DefaultBackground)
		}
	})

	t.Run("ResolveDefaultColor named", func(t *testing.T) {
		cases := []struct {
			name     int
			fg       bool
			expected color.RGBA
		}{
			{NamedColorForeground, true, DefaultForeground},
			{NamedColorBackground, false, DefaultBackground},
			{NamedColorCursor, true, DefaultCursorColor},
			{NamedColorBrightForeground, true, DefaultPalette[15]},
			{NamedColorDimForeground, true, color.RGBA{
				R: uint8(float64(DefaultForeground.R) * 0.66),
				G: uint8(float64(DefaultForeground.G) * 0.66),
				B: uint8(float64(DefaultForeground.B) * 0.66),
				A: 255,
			}},
			{0, true, DefaultPalette[0]},
			{15, true, DefaultPalette[15]},
		}
		for _, c := range cases {
			got := ResolveDefaultColor(&NamedColor{Name: c.name}, c.fg)
			if got != c.expected {
				t.Errorf("ResolveDefaultColor(NamedColor{Name:%d}, %v) = %+v, want %+v", c.name, c.fg, got, c.expected)
			}
		}
	})

	t.Run("ResolveDefaultColor named dim", func(t *testing.T) {
		for base := 0; base < 8; base++ {
			name := NamedColorDimBlack + base
			baseCol := DefaultPalette[base]
			want := color.RGBA{
				R: uint8(float64(baseCol.R) * 0.66),
				G: uint8(float64(baseCol.G) * 0.66),
				B: uint8(float64(baseCol.B) * 0.66),
				A: 255,
			}
			got := ResolveDefaultColor(&NamedColor{Name: name}, true)
			if got != want {
				t.Errorf("ResolveDefaultColor(NamedColor{Name:%d}) = %+v, want %+v", name, got, want)
			}
		}
	})

	t.Run("ResolveDefaultColor named out of range", func(t *testing.T) {
		got := ResolveDefaultColor(&NamedColor{Name: 999}, true)
		if got != DefaultForeground {
			t.Errorf("ResolveDefaultColor(999, true) = %+v, want %+v", got, DefaultForeground)
		}
		got = ResolveDefaultColor(&NamedColor{Name: 999}, false)
		if got != DefaultBackground {
			t.Errorf("ResolveDefaultColor(999, false) = %+v, want %+v", got, DefaultBackground)
		}
	})

	t.Run("ResolveDefaultColor fallback", func(t *testing.T) {
		c := color.NRGBA{R: 255, G: 128, B: 64, A: 255}
		got := ResolveDefaultColor(c, true)
		want := color.RGBA{R: 255, G: 128, B: 64, A: 255}
		if got != want {
			t.Errorf("ResolveDefaultColor(NRGBA) = %+v, want %+v", got, want)
		}
	})
}
