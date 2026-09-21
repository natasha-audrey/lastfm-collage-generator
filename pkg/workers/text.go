package workers

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"
	"sync"

	"github.com/go-text/render"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	"natasha-audrey/lastfm-collage-generator/static"
)

const labelSize = 16

var fontFiles = []string{
	"IBMPlexMono-Text.ttf", "fonts/NotoSans.ttf", "fonts/NotoSansCJKjp-Regular.otf",
	"fonts/NotoSansArabic.ttf", "fonts/NotoSansHebrew.ttf", "fonts/NotoSansDevanagari.ttf",
	"fonts/NotoSansThai.ttf", "fonts/NotoEmoji.ttf", "fonts/NotoSansSymbols2.ttf",
}

// Parse immutable font data once; each renderer gets its own mutable face caches.
var loadLabelFonts = sync.OnceValues(func() ([]*font.Font, error) {
	fonts := make([]*font.Font, 0, len(fontFiles))
	for _, name := range fontFiles {
		data, err := static.Fonts.ReadFile(name)
		if err != nil {
			return nil, err
		}
		face, err := font.ParseTTF(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("parse font %s: %w", name, err)
		}
		fonts = append(fonts, face.Font)
	}
	return fonts, nil
})

type labelFonts struct {
	faces  []*font.Face
	script language.Script
}

func (f *labelFonts) SetScript(script language.Script) { f.script = script }

func (f *labelFonts) ResolveFace(r rune) *font.Face {
	// Keep script-specific marks and punctuation with the letters they accompany.
	preferred := -1
	switch f.script {
	case language.Arabic:
		preferred = 3
	case language.Hebrew:
		preferred = 4
	case language.Devanagari:
		preferred = 5
	case language.Thai:
		preferred = 6
	}
	if r >= 0x1f000 || (r >= 0x2600 && r <= 0x27bf) {
		preferred = 7
	}
	if preferred >= 0 {
		if _, ok := f.faces[preferred].NominalGlyph(r); ok {
			return f.faces[preferred]
		}
	}
	for _, face := range f.faces {
		if _, ok := face.NominalGlyph(r); ok {
			return face
		}
	}
	// Uncovered characters remain visible as the primary font's missing glyph.
	return f.faces[0]
}

type labelRenderer struct {
	fonts     labelFonts
	segmenter shaping.Segmenter
	shaper    shaping.HarfbuzzShaper
}

func newLabelRenderer() (*labelRenderer, error) {
	fonts, err := loadLabelFonts()
	if err != nil {
		return nil, err
	}
	r := &labelRenderer{}
	for _, f := range fonts {
		face := font.NewFace(f)
		face.SetVariations([]font.Variation{{Tag: opentype.MustNewTag("wght"), Value: 400}})
		r.fonts.faces = append(r.fonts.faces, face)
	}
	return r, nil
}

// layout wraps shaped glyph clusters to the available pixel width in visual order.
func (r *labelRenderer) layout(label string, width int) []shaping.Line {
	if width <= 0 {
		return nil
	}
	var lines []shaping.Line
	for _, paragraph := range strings.Split(strings.ReplaceAll(label, "\r\n", "\n"), "\n") {
		text := []rune(paragraph)
		if len(text) == 0 {
			lines = append(lines, nil)
			continue
		}
		input := shaping.Input{Text: text, RunEnd: len(text), Size: fixed.I(labelSize)}
		runs := r.segmenter.Split(input, &r.fonts)
		outputs := make([]shaping.Output, len(runs))
		for i, run := range runs {
			outputs[i] = r.shaper.Shape(run)
		}
		var wrapper shaping.LineWrapper
		wrapped, _ := wrapper.WrapParagraph(shaping.WrapConfig{Direction: outputs[0].Direction}, width, text, shaping.NewSliceIterator(outputs))
		for _, line := range wrapped {
			sort.Slice(line, func(i, j int) bool { return line[i].VisualIndex < line[j].VisualIndex })
			lines = append(lines, line)
		}
	}
	return lines
}

func lineMetrics(line shaping.Line) (ascent, descent int) {
	ascent, descent = labelSize, 4
	for _, run := range line {
		ascent = max(ascent, run.LineBounds.Ascent.Ceil(), run.GlyphBounds.Ascent.Ceil())
		descent = max(descent, (-run.LineBounds.Descent).Ceil(), (-run.GlyphBounds.Descent).Ceil())
	}
	return
}

// drawLabels uses the same layout for the shadow and foreground, advancing by
// each line's actual height so wrapped artist names cannot overlap album titles.
func drawLabels(dst *image.RGBA, labels []string) error {
	r, err := newLabelRenderer()
	if err != nil {
		return err
	}
	painter := render.Renderer{FontSize: labelSize}
	y := 10
	for _, label := range labels {
		for _, line := range r.layout(label, dst.Bounds().Dx()-22) {
			ascent, descent := lineMetrics(line)
			y += ascent
			if y+descent+1 > dst.Bounds().Dy()-10 {
				return nil
			}
			for _, pass := range []struct {
				offset int
				color  color.Color
			}{{0, color.Black}, {1, color.White}} {
				painter.Color = pass.color
				x := fixed.I(10 + pass.offset)
				for _, run := range line {
					painter.DrawShapedRunAt(run, dst, x.Round(), y+pass.offset)
					x += run.Advance
				}
			}
			y += descent + 2
		}
	}
	return nil
}
