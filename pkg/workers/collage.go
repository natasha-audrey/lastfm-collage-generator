package workers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"

	"image/png"
	"io"
	"natasha-audrey/lastfm-collage-generator/pkg/logging"
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

// DownloadError indicates an artwork request failed.
type DownloadError struct{ Err error }

func (e *DownloadError) Error() string { return e.Err.Error() }
func (e *DownloadError) Unwrap() error { return e.Err }

type artworkBody struct{ io.ReadCloser }

func (b artworkBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil && err != io.EOF {
		err = &DownloadError{err}
	}
	return n, err
}

func downloadImages(albums []model.Album) error {
	return downloadImagesContext(context.Background(), http.DefaultClient, albums)
}

func downloadImagesContext(ctx context.Context, client *http.Client, albums []model.Album) error {
	for _, album := range albums {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := prepareAlbum(ctx, client, album)
		// Cancellation must stop generation rather than turn into a placeholder.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var download *DownloadError
		if errors.As(err, &download) {
			artworkWarning(ctx, album, "download_failed", err)
			_, err = addTextContext(ctx, album, []string{album.Artist, album.Name}, nil)
		}
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}

func prepareAlbum(ctx context.Context, client *http.Client, album model.Album) error {
	if album.Image == "" {
		artworkWarning(ctx, album, "missing_url", nil)
		_, err := addTextContext(ctx, album, []string{album.Artist, album.Name}, nil)
		return err
	}
	if _, err := os.Stat(album.LocalImage + ".png"); err == nil {
		logging.FromContext(ctx).DebugContext(ctx, "Artwork cache hit", "album", album.Name, "artist", album.Artist)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	logging.FromContext(ctx).DebugContext(ctx, "Downloading artwork", "album", album.Name, "artist", album.Artist)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, album.Image, nil)
	if err != nil {
		return &DownloadError{err}
	}
	response, err := client.Do(req)
	if err != nil {
		return &DownloadError{err}
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return &DownloadError{fmt.Errorf("artwork status %d", response.StatusCode)}
	}
	_, err = addTextContext(ctx, album, []string{album.Artist, album.Name}, artworkBody{response.Body})
	closeErr := response.Body.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func drawGradient(dst *image.RGBA) error {
	fp, err := os.Open("./static/black-gradient.png")
	if err != nil {
		return err
	}
	defer fp.Close()
	img, err := png.Decode(fp)
	if err != nil {
		return err
	}
	draw.Draw(dst, dst.Bounds(), img, image.Point{}, draw.Over)
	return nil
}

func addText(album model.Album, labels []string,
	body io.ReadCloser) (string, error) {

	return addTextContext(context.Background(), album, labels, body)
}

func artworkWarning(ctx context.Context, album model.Album, reason string, err error) {
	args := []any{"album", album.Name, "artist", album.Artist, "reason", reason}
	if err != nil {
		args = append(args, "error", err)
	}
	logging.FromContext(ctx).WarnContext(ctx, "Artwork fallback", args...)
}

func addTextContext(ctx context.Context, album model.Album, labels []string, body io.ReadCloser) (string, error) {
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
		albumImage, _, err := image.Decode(outFile)
		bg = albumImage
		if err != nil {
			bg = nil
			artworkWarning(ctx, album, "decode_failed", err)
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
	if err := drawGradient(rgba); err != nil {
		return "", err
	}
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
	return c.MakeCollageContext(context.Background(), albums, size, name)
}

// MakeCollageContext carries cancellation and attempt logging through collage generation.
func (c Collage) MakeCollageContext(ctx context.Context, albums []model.Album, size int, name string) (image.Image, error) {
	prepare := c.Prepare
	if prepare == nil {
		prepare = func(albums []model.Album) error { return downloadImagesContext(ctx, http.DefaultClient, albums) }
	}
	load := c.Load
	if load == nil {
		load = loadAlbumImage
	}
	save := c.Save
	if save == nil {
		save = saveCollage
	}

	done := logging.Stage(ctx, "artwork_preparation")
	err := prepare(albums)
	done()
	if err != nil {
		return nil, err
	}
	done = logging.Stage(ctx, "composition")
	result, err := composeCollage(albums, size, load)
	done()
	if err != nil {
		return nil, err
	}
	done = logging.Stage(ctx, "png_writing")
	err = save(name, result)
	done()
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Render prepares cached artwork and composes an image without saving a collage.
// Failed artwork downloads use labeled black tiles. Cancellation is checked
// between tiles and propagated to artwork downloads.
func Render(ctx context.Context, client *http.Client, albums []model.Album, size int) (image.Image, error) {
	if len(albums) > size*size && size > 0 {
		albums = albums[:size*size]
	}
	done := logging.Stage(ctx, "artwork_preparation")
	err := downloadImagesContext(ctx, client, albums)
	done()
	if err != nil {
		return nil, err
	}
	done = logging.Stage(ctx, "composition")
	defer done()
	return composeCollage(albums, size, func(album model.Album) (image.Image, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return loadAlbumImage(album)
	})
}
