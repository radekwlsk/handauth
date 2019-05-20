package main

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"gocv.io/x/gocv"
	"golang.org/x/image/draw"
	"gonum.org/v1/plot/palette/brewer"
	"image"
	"image/color"
	imgdraw "image/draw"
	"image/png"
	"log"
	"os"
	"path"
	"time"
)

const TestStartTimeFormat = "20060201-150405"

var (
	userId         int
	start          time.Time
	drawMassCenter bool
	samplesUsers   map[int][]int
)

type mask struct {
	img image.Image
}

func (m *mask) ColorModel() color.Model {
	return color.AlphaModel
}

func (m *mask) Bounds() image.Rectangle {
	return m.img.Bounds()
}

var black = color.Gray{Y: 0xff}

func (m *mask) At(x, y int) color.Color {
	if m.img.At(x, y) == black {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

func drawDot(dst *image.RGBA, center image.Point, c color.RGBA) {
	for _, x := range []int{center.X - 1, center.X, center.X + 1} {
		for _, y := range []int{center.Y - 1, center.Y, center.Y + 1} {
			p := image.Point{X: x, Y: y}
			if p.In(dst.Bounds()) {
				dst.SetRGBA(p.X, p.Y, c)
			}
		}
	}
}

var blackRGBA = color.RGBA{A: 255}
var redRGBA = color.RGBA{R: 255, A: 255}

func mergedSamples(id int) *image.RGBA {
	sample, err := cmd.ReadUserSample(uint16(id), uint16(id), uint8(samplesUsers[id][0]))
	if err != nil {
		panic(err)
	}
	sample.Preprocess()
	sampleMat := sample.Sample().Mat()
	img, err := sampleMat.ToImage()
	if err != nil {
		panic(err)
	}
	dst := image.NewRGBA(img.Bounds())
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: blackRGBA}, image.ZP, draw.Src)

	palSize := len(samplesUsers[id])
	if palSize > 12 {
		palSize = 12
	}
	pal, err := brewer.GetPalette(brewer.TypeQualitative, "Set3", palSize)
	if err != nil {
		log.Fatal(err)
	}
	colors := pal.Colors()

	var massCenters []image.Point
	for i, sampleId := range samplesUsers[id] {
		sample, err := cmd.ReadUserSample(uint16(id), uint16(id), uint8(sampleId))
		if err != nil {
			panic(err)
		}
		sample.Preprocess()
		var sampleMat gocv.Mat
		if drawMassCenter {
			massCenters = append(massCenters, sample.Sample().CenterOfMass())
		}
		sampleMat = sample.Sample().Mat()
		src, err := sampleMat.ToImage()
		if err != nil {
			panic(err)
		}
		if dst.Bounds().In(src.Bounds()) {
			newDst := image.NewRGBA(src.Bounds())
			draw.Draw(newDst, newDst.Bounds(), &image.Uniform{C: blackRGBA}, image.ZP, draw.Src)
			draw.Draw(newDst, dst.Bounds(), dst, image.ZP, draw.Src)
			dst = newDst
		}
		c := colors[i%12]
		msk := mask{src}
		imgdraw.DrawMask(dst, src.Bounds(), &image.Uniform{C: c}, image.ZP, &msk, image.ZP, imgdraw.Over)
	}
	for _, cm := range massCenters {
		drawDot(dst, cm, redRGBA)
	}
	return dst
}

func main() {
	flag.IntVar(&userId, "u", 1, "user to generate variation within user's samples")
	flag.BoolVar(&drawMassCenter, "cm", false, "draw center of mass of the samples")
	flag.Parse()

	start = time.Now()

	samplesUsers = cmd.GenuineUsers(true)
	if _, ok := samplesUsers[userId]; userId > 0 && !ok {
		log.Fatalf("No user %03d", userId)
	}

	start := time.Now()
	mergedFile, err := os.Create(path.Join("res", fmt.Sprintf("user%03d.png", userId)))
	mergedImage := mergedSamples(userId)
	if err != nil {
		log.Fatalf("failed to create: %s", err)
	}
	_ = png.Encode(mergedFile, mergedImage)
	defer mergedFile.Close()
	elapsed := time.Since(start)
	if flags.Verbose() {
		log.Printf("Merged samples from user %03d in %s\n", userId, elapsed)
	}
}
