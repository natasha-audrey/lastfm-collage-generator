package handler

import (
	"bufio"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/nathanyocum/lastfm-collage-generator/app/model"

	"github.com/golang/freetype"
	"golang.org/x/image/font"
)

var (
	dpi      = 72.0
	fontfile = "./web/static/fonts/IBMPlexMono-Text.ttf"
	size     = 16.0
	spacing  = 1.2
)

// AddText adds text at given x and y position with a given label
func AddText(fileName string, x, y int, labels []string, body io.ReadCloser) {
	outFile, err := os.Create(fileName)
	if body != nil {
		_, err = io.Copy(outFile, body)
		if err != nil {
			// log.Fatal(err)
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

	ptBlack := freetype.Pt(10, 10+int(c.PointToFixed(size)>>6))
	ptWhite := freetype.Pt(11, 11+int(c.PointToFixed(size)>>6))
	for _, label := range labels {

		fg = image.Black
		c.SetSrc(fg)
		if len(label) > 29 {
			_, err = c.DrawString(string(label[0:28]), ptBlack)
			ptBlack.Y += c.PointToFixed(size)
			_, err = c.DrawString(string(label[28:len(label)]), ptBlack)
		} else {
			_, err = c.DrawString(label, ptBlack)
		}
		fg = image.White
		c.SetSrc(fg)
		if len(label) > 29 {
			_, err = c.DrawString(string(label[0:28]), ptWhite)
			ptWhite.Y += c.PointToFixed(size)
			_, err = c.DrawString(string(label[28:len(label)]), ptWhite)
		} else {
			_, err = c.DrawString(label, ptWhite)
		}
		if err != nil {
			log.Println(err)
			return
		}
		ptBlack.Y += c.PointToFixed(size * spacing)
		ptWhite.Y += c.PointToFixed(size * spacing)
	}

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer outFile.Close()
	b := bufio.NewWriter(outFile)
	err = png.Encode(b, rgba)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = b.Flush()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// MakeCollage makes a collage of albums given an array of albums
func MakeCollage(albums []model.Album, size int) (im image.Image, err error) {
	bg := image.Black
	imageToReturn := image.NewRGBA(image.Rect(0, 0, 300*size, 300*size))
	draw.Draw(imageToReturn, imageToReturn.Bounds(), bg, image.ZP, draw.Src)
	xPos := 0
	yPos := 0
	hasNoImage := false
	for i := 0; i < size*size && i < len(albums); i++ {
		if albums[i].LocalImage != "" {
			file, err := os.Open(albums[i].LocalImage)
			if err != nil {
				if albums[i].Image == "" {
					hasNoImage = true
				}
				// return nil, errors.New("Could not open" + albums[i].LocalImage)
			}

			file.Seek(0, 0)
			if !hasNoImage {
				tempImage, err := png.Decode(file)
				if err != nil {
					return nil, errors.New("Could not decode image")
				}
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
