package workers

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"log/slog"
	"natasha-audrey/lastfm-collage-generator/pkg/model"
	"net/http"
	"os"
	"regexp"

	"github.com/golang/freetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var (
	dpi      = 72.0
	fontfile = "./static/IBMPlexMono-Text.ttf"
	size     = 16.0
	spacing  = 1.2
)

type Collage struct{}

func downloadImages(albums []model.Album, ch chan string) error {
	for _, album := range albums {
		if album.Image != "" {
			// If image exists don't bother making a new one
			if _, err := os.Stat(album.LocalImage + ".png"); os.IsNotExist(err) {
				slog.Info(album.LocalImage + ".png not found: Fetching " + album.Image)
				response, err := http.Get(album.Image)
				if err != nil {
					close(ch)
					log.Println(err)
					return err
				}

				file, err := addText(
					album,
					[]string{album.Artist, album.Name},
					response.Body)
				response.Body.Close()

				if err != nil {
					close(ch)
					log.Println("An error occurred:", err)
					return err
				}
				ch <- file
			} else {
				slog.Info(album.LocalImage + ".png already exists, skipping fetch.")
				ch <- album.LocalImage
			}
		} else {
			file, err := addText(album, []string{album.Artist, album.Name}, nil)
			if err != nil {
				close(ch)
				return err
			}
			ch <- file
		}
	}
	close(ch)
	return nil
}

func writeText(fg *image.Uniform, label string,
	c *freetype.Context, pt fixed.Point26_6) {

	c.SetSrc(fg)
	if len(label) > 29 {
		re := regexp.MustCompile(`.*\s`)
		lb := re.FindStringSubmatch(label[0:28])
		if lb[0] != "" {
			writeText(fg, string(label[0:len(lb[0])]), c, pt)
			writeText(fg, string(label[len(lb[0]):]), c,
				fixed.Point26_6{X: pt.X, Y: pt.Y + c.PointToFixed(size)})
			return
		}
	}
	_, err := c.DrawString(label, pt)
	if err != nil {
		log.Println(err)
	}
}

func drawGradient(dst *image.RGBA) {
	fp, _ := os.Open("./static/black-gradient.png")
	fp.Seek(0, 0)
	img, _ := png.Decode(fp)
	draw.Draw(dst, dst.Bounds(), img, image.Point{}, draw.Over)
}

// TODO(Refactor): Move Decode and Encode into their own files
func addText(album model.Album, labels []string,
	body io.ReadCloser) (string, error) {

	outFile, err := os.Create(album.LocalImage + ".png")
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	if body != nil {
		_, err = io.Copy(outFile, body)
		if err != nil {
			log.Println(album.LocalImage+album.Ext, err)
			return "", nil
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
			log.Println(album.LocalImage, err)
		}
	}
	if bg == nil {
		bg = image.Black
	}

	// Read the font data.
	fontBytes, err := os.ReadFile(fontfile)
	if err != nil {
		log.Println(album.LocalImage+".png", err)
		return "", nil
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(album.LocalImage+".png", err)
		return "", nil
	}

	// Initialize the context.
	fg := image.Black
	// rgba := body == nil ? image.NewRGBA(image.Rect(0, 0, 300, 300))  : image.NewRGBA(image.Rect(0, 0, bg.Bounds().Dx(), bg.Bounds().Dy()))
	var rgba *image.RGBA
	if body == nil {
		rgba = image.NewRGBA(image.Rect(0, 0, 300, 300))
	} else {
		rgba = image.NewRGBA(image.Rect(0, 0, bg.Bounds().Dx(), bg.Bounds().Dy()))
	}
	draw.Draw(rgba, rgba.Bounds(), bg, image.Point{}, draw.Src)
	drawGradient(rgba)

	c := freetype.NewContext()
	c.SetDPI(dpi)
	c.SetFont(f)
	c.SetFontSize(size)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)
	c.SetHinting(font.HintingFull)

	// Save that RGBA image to disk.
	outFile, err = os.Create(album.LocalImage + ".png")
	if err != nil {
		log.Println(album.LocalImage+".png", err)
		return "", err
	}

	ptBlack := freetype.Pt(10, 10+int(c.PointToFixed(size)>>6))
	ptWhite := freetype.Pt(11, 11+int(c.PointToFixed(size)>>6))
	for _, label := range labels {
		writeText(image.Black, label, c, ptBlack)
		writeText(image.White, label, c, ptWhite)
		ptBlack.Y += c.PointToFixed(size * spacing)
		ptWhite.Y += c.PointToFixed(size * spacing)
	}
	defer outFile.Close()
	b := bufio.NewWriter(outFile)
	err = png.Encode(b, rgba)
	if err != nil {
		log.Println(album.LocalImage+".png", err)
		return "", err
	}
	err = b.Flush()
	if err != nil {
		log.Println(album.LocalImage+".png", err)
		return "", err
	}
	if body != nil {
		body.Close()
	}
	return album.LocalImage + ".png", nil
}

// MakeCollage makes a collage of albums given an array of albums
func (c Collage) MakeCollage(albums []model.Album, size int, name string) (im image.Image, err error) {
	ch := make(chan string)
	go downloadImages(albums, ch)
	for v := range ch {
		if v == "" {
			a := fmt.Sprintf("Error generating %s\n", v)
			panic(a)
		}
	}

	bg := image.Black
	imageToReturn := image.NewRGBA(image.Rect(0, 0, 300*size, 300*size))
	draw.Draw(imageToReturn, imageToReturn.Bounds(), bg, image.Point{}, draw.Src)
	xPos := 0
	yPos := 0

	for i := 0; i < size*size && i < len(albums); i++ {
		if albums[i].LocalImage != "" {
			file, err := os.Open(albums[i].LocalImage + ".png")
			if err != nil {
				log.Println(err)
				// return nil, err
			}

			file.Seek(0, 0)
			tempImage, err := png.Decode(file)
			if err != nil {
				fmt.Println(err)
				_, err = jpeg.Decode(file)
				if err != nil {
					// Some kind of error happened, regenerate the image without an album
					log.Println("Error getting images", albums[i].LocalImage)
					addText(albums[i], []string{albums[i].Artist, albums[i].Name}, nil)
					i--
				}
			} else {
				tempPoint := image.Point{xPos, yPos}
				tempRect := image.Rectangle{tempPoint, tempPoint.Add(tempImage.Bounds().Size())}
				draw.Draw(imageToReturn, tempRect, tempImage, image.Point{}, draw.Src)
				xPos += tempImage.Bounds().Dx()
				if (i+1)%size == 0 {
					xPos = 0
					yPos += tempImage.Bounds().Dy()
				}
			}
		}
	}
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, imageToReturn)
	return imageToReturn, nil
}
