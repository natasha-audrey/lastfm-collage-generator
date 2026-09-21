package workers

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

func TestLabelUnicodeCoverage(t *testing.T) {
	r, err := newLabelRenderer()
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{
		"Björk — Jóga", "WHAT WE DREW 우리가 그려왔던", "Cafe\u0301", "宇多田ヒカル 東京の夜", "周杰倫", "방탄소년단",
		"Привет κόσμε", "موسيقى عربية", "שלום עולם", "नमस्ते दुनिया", "สวัสดี", "😀 👩‍🚀 🏳️‍🌈", "⏵ ⏹ ♫",
	} {
		t.Run(label, func(t *testing.T) {
			lines := r.layout(label, 278)
			if len(lines) == 0 {
				t.Fatal("no lines")
			}
			for _, line := range lines {
				for _, run := range line {
					for _, glyph := range run.Glyphs {
						if glyph.GlyphID == 0 {
							t.Errorf("missing glyph at rune %d", glyph.TextIndex())
						}
					}
				}
			}
		})
	}
}

func TestLabelWrapping(t *testing.T) {
	r, err := newLabelRenderer()
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{strings.Repeat("a", 90), strings.Repeat("東京の夜", 12), strings.Repeat("Cafe\u0301 ", 15), strings.Repeat("👩‍🚀", 30)} {
		lines := r.layout(label, 100)
		if len(lines) < 2 {
			t.Fatalf("%q did not wrap", label)
		}
		count := 0
		for _, line := range lines {
			var width fixed.Int26_6
			for _, run := range line {
				width += run.Advance
				count += run.Runes.Count
			}
			if width > fixed.I(100) {
				t.Errorf("line width %v exceeds 100px", width)
			}
		}
		if count != len([]rune(label)) {
			t.Errorf("lost runes: got %d, want %d", count, len([]rune(label)))
		}
	}
	if got := r.layout("", 278); len(got) != 1 {
		t.Fatal("empty label should reserve one line")
	}
	if got := r.layout("one\ntwo", 278); len(got) != 2 {
		t.Fatal("explicit newline was not preserved")
	}
	if got := r.layout("text", 0); len(got) != 0 {
		t.Fatal("zero width should not render")
	}
}

func TestLabelShaping(t *testing.T) {
	r, err := newLabelRenderer()
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"👩‍🚀", "🏳️‍🌈"} {
		lines := r.layout(label, 278)
		glyphs := 0
		for _, line := range lines {
			for _, run := range line {
				for _, g := range run.Glyphs {
					if g.Width != 0 && g.Height != 0 {
						glyphs++
					}
				}
			}
		}
		if glyphs != 1 {
			t.Errorf("%q: got %d glyphs, want one joined glyph", label, glyphs)
		}
	}
	// Arabic letters must use contextual forms rather than isolated nominal glyphs.
	arabic := []rune("مرحبا")
	contextual := false
	for _, line := range r.layout(string(arabic), 278) {
		for _, run := range line {
			for _, glyph := range run.Glyphs {
				nominal, _ := run.Face.NominalGlyph(arabic[glyph.TextIndex()])
				if glyph.GlyphID != nominal {
					contextual = true
				}
			}
		}
	}
	if !contextual {
		t.Fatal("Arabic letters were not contextually shaped")
	}
	lines := r.layout("hello مرحبا world", 278)
	rtl := false
	for _, line := range lines {
		for i, run := range line {
			if i > 0 && run.VisualIndex < line[i-1].VisualIndex {
				t.Fatal("runs not in visual order")
			}
			if run.Direction.Progression() != lines[0][0].Direction.Progression() {
				rtl = true
			}
		}
	}
	if !rtl {
		t.Fatal("mixed text has no RTL run")
	}
}

func TestLabelsAdvancePastWrappedArtist(t *testing.T) {
	r, err := newLabelRenderer()
	if err != nil {
		t.Fatal(err)
	}
	artist := strings.Repeat("Long Artist ", 5)
	first := image.NewRGBA(image.Rect(0, 0, 300, 300))
	both := image.NewRGBA(first.Bounds())
	if err := drawLabels(first, []string{artist}); err != nil {
		t.Fatal(err)
	}
	if err := drawLabels(both, []string{artist, "Album"}); err != nil {
		t.Fatal(err)
	}
	y := 10
	for _, line := range r.layout(artist, 278) {
		a, d := lineMetrics(line)
		y += a + d + 2
	}
	changes := 0
	for py := 0; py < 300; py++ {
		for px := 0; px < 300; px++ {
			if first.RGBAAt(px, py) != both.RGBAAt(px, py) {
				if py < y {
					t.Fatalf("album overlaps artist at y=%d (next block starts %d)", py, y)
				}
				changes++
			}
		}
	}
	if changes == 0 {
		t.Fatal("album label was not drawn")
	}
}

// TestLabelPreview optionally writes a visual fixture without requiring Last.fm.
func TestLabelPreview(t *testing.T) {
	path := os.Getenv("COLLAGE_TEXT_PREVIEW")
	if path == "" {
		t.Skip("set COLLAGE_TEXT_PREVIEW to export a preview")
	}
	img := image.NewRGBA(image.Rect(0, 0, 900, 600))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{30, 30, 30, 255}), image.Point{}, draw.Src)
	labels := [][]string{{"Björk — Cafe\u0301", "An Album With A Really Really Long Title"}, {"宇多田ヒカル", "東京の夜に聴く音楽東京の夜に聴く音楽"}, {"방탄소년단", "周杰倫 — 夜曲"}, {"موسيقى عربية", "hello مرحبا world"}, {"नमस्ते दुनिया", "שלום עולם — สวัสดี"}, {"👩‍🚀 🏳️‍🌈 😀", strings.Repeat("a", 60)}}
	for i, pair := range labels {
		tile := image.NewRGBA(image.Rect(0, 0, 300, 300))
		draw.Draw(tile, tile.Bounds(), image.NewUniform(color.RGBA{30, 30, 30, 255}), image.Point{}, draw.Src)
		if err := drawLabels(tile, pair); err != nil {
			t.Fatal(err)
		}
		at := image.Pt((i%3)*300, (i/3)*300)
		draw.Draw(img, image.Rectangle{Min: at, Max: at.Add(image.Pt(300, 300))}, tile, image.Point{}, draw.Src)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

var _ shaping.FontmapScript = (*labelFonts)(nil)
