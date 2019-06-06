package handler

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/golang/freetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/nathanyocum/lastfm-collage-generator/app/model"
)

var (
	dpi      = 72.0
	fontfile = "./web/static/fonts/IBMPlexMono-Text.ttf"
	size     = 16.0
	spacing  = 1.2
)

func writeText(fg *image.Uniform, label string,
	c *freetype.Context, pt fixed.Point26_6) {

	c.SetSrc(fg)
	if len(label) > 29 {
		re := regexp.MustCompile(`.*\s`)
		lb := re.FindStringSubmatch(label[0:28])
		if lb[0] != "" {
			writeText(fg, string(label[0:len(lb[0])]), c, pt)
			writeText(fg, string(label[len(lb[0]):len(label)]), c,
				fixed.Point26_6{X: pt.X, Y: pt.Y + c.PointToFixed(size)})
			return
		}
	}
	_, err := c.DrawString(label, pt)
	if err != nil {
		log.Println(err)
	}
}

// AddText adds text at given x and y position with a given label
func AddText(fileName string, x, y int, labels []string,
	body io.ReadCloser) {

	outFile, err := os.Create(fileName)
	if body != nil {
		_, err = io.Copy(outFile, body)
		if err != nil {
			log.Println(err)
			return
		}
	}

	outFile.Seek(0, 0)
	var bg image.Image
	if body != nil {
		bg, err = jpeg.Decode(outFile)
		if err != nil {
			outFile.Seek(0, 0)
			bg, err = png.Decode(outFile)
			if err != nil {
				log.Println(err)
				return
			}
		}
	} else {
		bg = image.Black
	}
	// Read the font data.
	fontBytes, err := ioutil.ReadFile(fontfile)
	if err != nil {
		log.Println(err)
		return
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
		return
	}

	// Initialize the context.
	fg := image.Black
	// rgba := body == nil ? image.NewRGBA(image.Rect(0, 0, 300, 300))  : image.NewRGBA(image.Rect(0, 0, bg.Bounds().Dx(), bg.Bounds().Dy()))
	var rgba draw.Image
	if body == nil {
		rgba = image.NewRGBA(image.Rect(0, 0, 300, 300))
	} else {
		rgba = image.NewRGBA(image.Rect(0, 0, bg.Bounds().Dx(), bg.Bounds().Dy()))
	}
	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)
	c := freetype.NewContext()
	c.SetDPI(dpi)
	c.SetFont(f)
	c.SetFontSize(size)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)
	c.SetHinting(font.HintingFull)

	// Save that RGBA image to disk.
	outFile, err = os.Create(fileName)
	if err != nil {
		log.Println(err)
		return
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
		log.Println(err)
		return
	}
	err = b.Flush()
	if err != nil {
		log.Println(err)
		return
	}
}

// MakeCollage makes a collage of albums given an array of albums
func MakeCollage(albums []model.Album, size int) (im image.Image, err error) {
	bg := image.Black
	imageToReturn := image.NewRGBA(image.Rect(0, 0, 300*size, 300*size))
	draw.Draw(imageToReturn, imageToReturn.Bounds(), bg, image.ZP, draw.Src)
	xPos := 0
	yPos := 0
	for i := 0; i < size*size && i < len(albums); i++ {
		if albums[i].LocalImage != "" {
			file, err := os.Open(albums[i].LocalImage)
			if err != nil {
				log.Println(err)
				return nil, err
			}

			file.Seek(0, 0)
			tempImage, err := png.Decode(file)
			if err != nil {
				tempImage, err = jpeg.Decode(file)
				if err != nil {
					// Some kind of error happened, regenerate the image without an album
					response, err := http.Get(albums[i].Image)
					if err != nil {
						fmt.Println("Error getting images")
						AddText(albums[i].LocalImage, 0, 0, []string{albums[i].Artist, albums[i].Name}, nil)
						i--
					}
					defer response.Body.Close()
					AddText(
						albums[i].LocalImage,
						0,
						0,
						[]string{albums[i].Artist, albums[i].Name},
						response.Body)
					i--
				}
			} else {
				tempPoint := image.Point{xPos, yPos}
				tempRect := image.Rectangle{tempPoint, tempPoint.Add(tempImage.Bounds().Size())}
				draw.Draw(imageToReturn, tempRect, tempImage, image.ZP, draw.Src)
				xPos += tempImage.Bounds().Dx()
				if (i+1)%size == 0 {
					xPos = 0
					yPos += tempImage.Bounds().Dy()
				}
			}
		}
	}
	return imageToReturn, nil
}
