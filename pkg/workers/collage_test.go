package workers

import (
	"errors"
	"image"
	"image/color"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/model"
)

func TestCollageMakeCollageComposesAndSavesImages(t *testing.T) {
	albums := []model.Album{{Name: "one"}, {Name: "two"}}
	images := map[string]image.Image{
		"one": solidImage(color.RGBA{R: 255, A: 255}),
		"two": solidImage(color.RGBA{G: 255, A: 255}),
	}
	var prepared []model.Album
	var savedName string
	var saved image.Image

	collage := Collage{
		Prepare: func(got []model.Album) error {
			prepared = got
			return nil
		},
		Load: func(album model.Album) (image.Image, error) {
			return images[album.Name], nil
		},
		Save: func(name string, got image.Image) error {
			savedName, saved = name, got
			return nil
		},
	}

	got, err := collage.MakeCollage(albums, 2, "collage.png")
	if err != nil {
		t.Fatalf("MakeCollage() error = %v", err)
	}
	if len(prepared) != len(albums) {
		t.Fatalf("Prepare received %d albums, want %d", len(prepared), len(albums))
	}
	if savedName != "collage.png" || saved != got {
		t.Fatal("MakeCollage did not save the generated collage")
	}
	if got.Bounds() != image.Rect(0, 0, 600, 600) {
		t.Fatalf("collage bounds = %v, want %v", got.Bounds(), image.Rect(0, 0, 600, 600))
	}
	if got.At(0, 0) != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("first album was not drawn at the first tile")
	}
	if got.At(300, 0) != (color.RGBA{G: 255, A: 255}) {
		t.Errorf("second album was not drawn at the second tile")
	}
	if got.At(0, 300) != (color.RGBA{A: 255}) {
		t.Errorf("unfilled tile = %v, want black", got.At(0, 300))
	}
}

func TestCollageMakeCollageReturnsDependencyErrors(t *testing.T) {
	want := errors.New("prepare failed")
	collage := Collage{
		Prepare: func([]model.Album) error { return want },
		Load: func(model.Album) (image.Image, error) {
			t.Fatal("Load must not run after Prepare fails")
			return nil, nil
		},
	}

	_, err := collage.MakeCollage([]model.Album{{}}, 1, "ignored.png")
	if !errors.Is(err, want) {
		t.Fatalf("MakeCollage() error = %v, want %v", err, want)
	}
}

func TestComposeCollageRejectsInvalidSize(t *testing.T) {
	_, err := composeCollage(nil, 0, nil)
	if err == nil {
		t.Fatal("composeCollage() error = nil, want an error")
	}
}

func solidImage(c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 300; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}
