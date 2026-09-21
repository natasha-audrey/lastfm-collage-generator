package workers

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/model"

	"github.com/golang/freetype"
)

func TestWriteTextReturnsDrawingError(t *testing.T) {
	for _, tc := range []struct {
		name  string
		label string
	}{
		{name: "single line", label: "Album"},
		{name: "wrapped line", label: "An Album With A Really Really Long Title"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A context without a font makes DrawString fail before rendering.
			ctx := freetype.NewContext()
			err := writeText(image.Black, tc.label, ctx, freetype.Pt(10, 10))
			const want = "freetype: DrawText called with a nil font"
			if err == nil || err.Error() != want {
				t.Fatalf("writeText() error = %v, want %q", err, want)
			}
		})
	}
}

type collageRoundTripper func(*http.Request) (*http.Response, error)

func (f collageRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type collageResponseBody struct {
	io.Reader
	closed bool
	err    error
}

func (b *collageResponseBody) Close() error {
	b.closed = true
	return b.err
}

func TestDownloadImages(t *testing.T) {
	// Rendering loads static assets relative to the repository root. Chdir and
	// DefaultClient are process-wide, so these tests must remain sequential.
	t.Chdir(filepath.Join("..", ".."))
	originalClient := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = originalClient })

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, solidImage(color.RGBA{R: 255, A: 255})); err != nil {
		t.Fatal(err)
	}
	getErr := errors.New("download failed")
	closeErr := errors.New("close failed")
	for _, tc := range []struct {
		name       string
		empty      bool
		cached     bool
		noImage    bool
		badPath    bool
		getErr     error
		closeErr   error
		invalidExt string
	}{
		{name: "empty input", empty: true},
		{name: "download and render"},
		{name: "cached image", cached: true},
		{name: "placeholder", noImage: true},
		{name: "invalid JPEG", invalidExt: ".jpg"},
		{name: "invalid JPEG alternate extension", invalidExt: ".jpeg"},
		{name: "invalid GIF", invalidExt: ".gif"},
		{name: "invalid PNG", invalidExt: ".png"},
		{name: "unsupported format", invalidExt: ".webp"},
		{name: "HTTP error", getErr: getErr},
		{name: "render error", badPath: true},
		{name: "placeholder error", noImage: true, badPath: true},
		{name: "close error", closeErr: closeErr},
		{name: "render error takes precedence over close error", badPath: true, closeErr: closeErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "album")
			if tc.badPath {
				path = filepath.Join(path, "missing", "album")
			}
			cached := []byte("existing image must remain untouched")
			if tc.cached {
				if err := os.WriteFile(path+".png", cached, 0600); err != nil {
					t.Fatal(err)
				}
			}
			album := model.Album{
				Artist:     "Artist",
				Name:       "An Album With A Really Really Long Title",
				Image:      "https://example.test/album.png",
				LocalImage: path,
				Ext:        ".png",
			}
			if tc.noImage {
				album.Image = ""
			}
			data := encoded.Bytes()
			if tc.invalidExt != "" {
				album.Ext = tc.invalidExt
				data = []byte("truncated artwork")
			}
			body := &collageResponseBody{Reader: bytes.NewReader(data), err: tc.closeErr}
			requests := 0
			http.DefaultClient = &http.Client{Transport: collageRoundTripper(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.Method != http.MethodGet || r.URL.String() != album.Image {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if tc.getErr != nil {
					return nil, tc.getErr
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body}, nil
			})}
			albums := []model.Album{album}
			if tc.empty {
				albums = nil
			} else {
				// Success must continue to the next album; errors must stop before it.
				albums = append(albums, model.Album{LocalImage: filepath.Join(t.TempDir(), "next")})
			}
			err := downloadImages(albums)
			switch {
			case tc.getErr != nil:
				if !errors.Is(err, tc.getErr) {
					t.Fatalf("error = %v, want %v", err, tc.getErr)
				}
			case tc.badPath:
				var pathErr *os.PathError
				if !errors.As(err, &pathErr) || pathErr.Path != path+".png" {
					t.Fatalf("error = %v, want output path error", err)
				}
			case tc.closeErr != nil:
				if !errors.Is(err, tc.closeErr) {
					t.Fatalf("error = %v, want %v", err, tc.closeErr)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
			}
			wantRequests := 1
			if tc.empty || tc.cached || tc.noImage {
				wantRequests = 0
			}
			if requests != wantRequests {
				t.Errorf("requests = %d, want %d", requests, wantRequests)
			}
			if wantClosed := wantRequests == 1 && tc.getErr == nil; body.closed != wantClosed {
				t.Errorf("body closed = %v, want %v", body.closed, wantClosed)
			}
			if tc.empty {
				return
			}
			if err != nil {
				if _, statErr := os.Stat(albums[1].LocalImage + ".png"); !os.IsNotExist(statErr) {
					t.Errorf("next album was processed after failure: %v", statErr)
				}
				return
			}
			for i, a := range albums {
				data, err := os.ReadFile(a.LocalImage + ".png")
				if err != nil {
					t.Fatal(err)
				}
				if i == 0 && tc.cached {
					if !bytes.Equal(data, cached) {
						t.Error("cached image was modified")
					}
					continue
				}
				img, err := png.Decode(bytes.NewReader(data))
				if err != nil {
					t.Fatalf("decode album %d: %v", i, err)
				}
				if tc.invalidExt != "" && color.RGBAModel.Convert(img.At(299, 299)) != (color.RGBA{A: 255}) {
					t.Errorf("album %d fallback background is not black", i)
				}
				if img.Bounds() != image.Rect(0, 0, 300, 300) {
					t.Errorf("album %d bounds = %v", i, img.Bounds())
				}
			}
		})
	}
}

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

func TestLoadAlbumImage(t *testing.T) {
	t.Run("valid PNG", func(t *testing.T) {
		want := image.NewRGBA(image.Rect(0, 0, 2, 1))
		want.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
		want.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
		var data bytes.Buffer
		if err := png.Encode(&data, want); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "album")
		if err := os.WriteFile(path+".png", data.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}

		got, err := loadAlbumImage(model.Album{LocalImage: path})
		if err != nil {
			t.Fatalf("loadAlbumImage() error = %v", err)
		}
		if got == nil {
			t.Fatal("loadAlbumImage() returned a nil image")
		}
		if got.Bounds() != want.Bounds() {
			t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
		}
		for x := 0; x < 2; x++ {
			if pixel := color.RGBAModel.Convert(got.At(x, 0)); pixel != want.RGBAAt(x, 0) {
				t.Errorf("pixel (%d, 0) = %v, want %v", x, pixel, want.RGBAAt(x, 0))
			}
		}
	})

	t.Run("empty path", func(t *testing.T) {
		got, err := loadAlbumImage(model.Album{})
		if err == nil || err.Error() != "album has no local image path" {
			t.Fatalf("loadAlbumImage() error = %v, want missing local path error", err)
		}
		if got != nil {
			t.Error("loadAlbumImage() returned an image on failure")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing")
		got, err := loadAlbumImage(model.Album{LocalImage: path})
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("loadAlbumImage() error = %v, want file not found", err)
		}
		var pathErr *os.PathError
		if !errors.As(err, &pathErr) || pathErr.Path != path+".png" {
			t.Errorf("loadAlbumImage() error = %v, want path %q", err, path+".png")
		}
		if got != nil {
			t.Error("loadAlbumImage() returned an image on failure")
		}
	})

	t.Run("invalid PNG", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid")
		if err := os.WriteFile(path+".png", []byte("not a PNG image"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := loadAlbumImage(model.Album{LocalImage: path})
		var formatErr png.FormatError
		if !errors.As(err, &formatErr) {
			t.Fatalf("loadAlbumImage() error = %v, want PNG format error", err)
		}
		if got != nil {
			t.Error("loadAlbumImage() returned an image on failure")
		}
	})
}

func TestSaveCollage(t *testing.T) {
	t.Run("valid PNG", func(t *testing.T) {
		want := image.NewRGBA(image.Rect(0, 0, 2, 1))
		want.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
		want.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
		path := filepath.Join(t.TempDir(), "collage.png")

		if err := saveCollage(path, want); err != nil {
			t.Fatalf("saveCollage() error = %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("decode saved collage: %v", err)
		}
		if got.Bounds() != want.Bounds() {
			t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
		}
		for x := 0; x < 2; x++ {
			if pixel := color.RGBAModel.Convert(got.At(x, 0)); pixel != want.RGBAAt(x, 0) {
				t.Errorf("pixel (%d, 0) = %v, want %v", x, pixel, want.RGBAAt(x, 0))
			}
		}
	})

	t.Run("file creation error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing", "collage.png")
		err := saveCollage(path, image.NewRGBA(image.Rect(0, 0, 1, 1)))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("saveCollage() error = %v, want file not found", err)
		}
		var pathErr *os.PathError
		if !errors.As(err, &pathErr) || pathErr.Path != path {
			t.Errorf("saveCollage() error = %v, want path %q", err, path)
		}
	})

	t.Run("PNG encoding error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "collage.png")
		// PNG cannot represent an image with zero width or height.
		err := saveCollage(path, image.NewRGBA(image.Rectangle{}))
		var formatErr png.FormatError
		if !errors.As(err, &formatErr) {
			t.Fatalf("saveCollage() error = %v, want PNG format error", err)
		}
	})
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
