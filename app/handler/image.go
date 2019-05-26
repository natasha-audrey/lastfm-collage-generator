package handler

import (
	"bufio"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/freetype"
	"golang.org/x/image/font"
)

var (
	dpi      = 72.0
	fontfile = "./web/fonts/IBMPlexMono-Text.ttf"
	size     = 10.0
	spacing  = 1.5
)

//AddText adds text at given x and y position with a given label
func AddText(fileName string, x, y int, labels []string, body io.ReadCloser) {
	outFile, err := os.Create(fileName)
	_, err = io.Copy(outFile, body)
	if err != nil {
		log.Fatal(err)
	}

	outFile.Seek(0, 0)
	bg, err := jpeg.Decode(outFile)
	if err != nil {
		outFile.Seek(0, 0)
		bg, err = png.Decode(outFile)
		if err != nil {
			log.Println(err)
			return
		}
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
	rgba := image.NewRGBA(image.Rect(0, 0, 174, 174))
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
	ptWhite := freetype.Pt(10, 11+int(c.PointToFixed(size)>>6))
	for _, label := range labels {

		fg = image.Black
		c.SetSrc(fg)
		if len(label) > 26 {
			_, err = c.DrawString(string(label[0:25]), ptBlack)
			ptBlack.Y += c.PointToFixed(size)
			_, err = c.DrawString(string(label[25:len(label)]), ptBlack)
		} else {
			_, err = c.DrawString(label, ptBlack)
		}
		fg = image.White
		c.SetSrc(fg)
		if len(label) > 26 {
			_, err = c.DrawString(string(label[0:25]), ptWhite)
			ptWhite.Y += c.PointToFixed(size)
			_, err = c.DrawString(string(label[25:len(label)]), ptWhite)
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
