package workers

import (
	"bufio"
	"errors"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"natasha-audrey/lastfm-collage-generator/pkg/model"
	"net/http"
	"os"
)

// Collage creates an album collage. The optional functions make the I/O at the
// edges replaceable; a zero-value Collage uses the filesystem and HTTP.
type Collage struct {
	// Prepare prepares artwork; nil downloads and labels images under their local paths.
	Prepare func([]model.Album) error
	// Load reads a tile; nil decodes the PNG at Album.LocalImage + ".png".
	Load func(model.Album) (image.Image, error)
	// Save writes the collage; nil creates or overwrites the named file as a PNG.
	Save func(string, image.Image) error
}

func downloadImages(albums []model.Album) error {
	for _, album := range albums {
		if album.Image != "" {
			// If image exists don't bother making a new one
			if _, err := os.Stat(album.LocalImage + ".png"); os.IsNotExist(err) {
				slog.Info(album.LocalImage + ".png not found: Fetching " + album.Image)
				response, err := http.Get(album.Image)
				if err != nil {
					return err
				}

				_, err = addText(
					album,
					[]string{album.Artist, album.Name},
					response.Body)
				closeErr := response.Body.Close()

				if err != nil {
					return err
				}
				if closeErr != nil {
					return closeErr
				}
			} else {
				slog.Info(album.LocalImage + ".png already exists, skipping fetch.")
			}
		} else {
			_, err := addText(album, []string{album.Artist, album.Name}, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func drawGradient(dst *image.RGBA) {
	fp, _ := os.Open("./static/black-gradient.png")
	fp.Seek(0, 0)
	img, _ := png.Decode(fp)
	draw.Draw(dst, dst.Bounds(), img, image.Point{}, draw.Over)
}

func addText(album model.Album, labels []string,
	body io.ReadCloser) (string, error) {

	outFile, err := os.Create(album.LocalImage + ".png")
	if err != nil {
		return "", err
	}
	defer func() { _ = outFile.Close() }()
	if body != nil {
		_, err = io.Copy(outFile, body)
		if err != nil {
			return "", err
		}
	}

	outFile.Seek(0, 0)
	// decode the file
	var bg image.Image
	if body != nil {
		if album.Ext == ".jpg" || album.Ext == ".jpeg" {
			bg, err = jpeg.Decode(outFile)
		}
		if album.Ext == ".gif" {
			bg, err = gif.Decode(outFile)
		}
		if album.Ext == ".png" {
			bg, err = png.Decode(outFile)
		}
		if err != nil {
			bg = nil
			slog.Info("Ran into problem decoding, using a black background image", album.LocalImage, err)
		}
	}
	// Uniform black has effectively infinite bounds, so size the fallback explicitly.
	bounds := image.Rect(0, 0, 300, 300)
	if bg == nil {
		bg = image.Black
	} else {
		bounds = image.Rect(0, 0, bg.Bounds().Dx(), bg.Bounds().Dy())
	}

	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, rgba.Bounds(), bg, image.Point{}, draw.Src)
	drawGradient(rgba)
	if err := drawLabels(rgba, labels); err != nil {
		return "", err
	}

	// Save that RGBA image to disk.
	if err := outFile.Close(); err != nil {
		return "", err
	}
	outFile, err = os.Create(album.LocalImage + ".png")
	if err != nil {
		return "", err
	}

	b := bufio.NewWriter(outFile)
	err = png.Encode(b, rgba)
	if err != nil {
		return "", err
	}
	err = b.Flush()
	if err != nil {
		return "", err
	}
	return album.LocalImage + ".png", nil
}

func loadAlbumImage(album model.Album) (image.Image, error) {
	if album.LocalImage == "" {
		return nil, errors.New("album has no local image path")
	}

	file, err := os.Open(album.LocalImage + ".png")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func saveCollage(name string, img image.Image) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func composeCollage(albums []model.Album, size int, load func(model.Album) (image.Image, error)) (image.Image, error) {
	if size <= 0 {
		return nil, errors.New("collage size must be positive")
	}

	result := image.NewRGBA(image.Rect(0, 0, 300*size, 300*size))
	draw.Draw(result, result.Bounds(), image.Black, image.Point{}, draw.Src)

	for i := 0; i < size*size && i < len(albums); i++ {
		albumImage, err := load(albums[i])
		if err != nil {
			return nil, err
		}

		position := image.Point{X: (i % size) * 300, Y: (i / size) * 300}
		rectangle := image.Rectangle{Min: position, Max: position.Add(albumImage.Bounds().Size())}
		draw.Draw(result, rectangle, albumImage, albumImage.Bounds().Min, draw.Src)
	}

	return result, nil
}

// MakeCollage prepares all albums, places up to size*size tiles in row order,
// and saves the result to name. The canvas is size*300 pixels on each side,
// with unused space filled black; artwork is not resized. Size must be positive.
// It returns the saved image, or an error from preparation, composition, or saving.
func (c Collage) MakeCollage(albums []model.Album, size int, name string) (image.Image, error) {
	prepare := c.Prepare
	if prepare == nil {
		prepare = downloadImages
	}
	load := c.Load
	if load == nil {
		load = loadAlbumImage
	}
	save := c.Save
	if save == nil {
		save = saveCollage
	}

	if err := prepare(albums); err != nil {
		return nil, err
	}
	result, err := composeCollage(albums, size, load)
	if err != nil {
		return nil, err
	}
	if err := save(name, result); err != nil {
		return nil, err
	}
	return result, nil
}
