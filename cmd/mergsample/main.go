package main

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
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
	userId       int
	start        time.Time
	startString  string
	samplesUsers map[int][]int
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

func mergedSamples(id int) *image.Gray {
	sample, err := cmd.ReadUserSample(uint8(id), uint8(id), uint8(samplesUsers[id][0]))
	if err != nil {
		panic(err)
	}
	ratio := sample.Sample().Ratio()
	sample.Preprocess()
	sampleMat := sample.Sample().Mat()
	img, err := sampleMat.ToImage()
	if err != nil {
		panic(err)
	}
	mergedImg := image.NewGray(img.Bounds())
	for _, sampleId := range samplesUsers[id] {
		sample, err := cmd.ReadUserSample(uint8(id), uint8(id), uint8(sampleId))
		if err != nil {
			panic(err)
		}
		sample.Sample().Preprocess(ratio)
		sampleMat := sample.Sample().Mat()
		img2, err := sampleMat.ToImage()
		if err != nil {
			panic(err)
		}
		imgdraw.DrawMask(mergedImg, img2.Bounds(), img2, image.ZP, &mask{img2}, img2.Bounds().Min, imgdraw.Over)
	}
	return mergedImg
}

func main() {
	flag.IntVar(&userId, "u", 1, "user to generate variation within user's samples")
	flag.Parse()

	start = time.Now()
	startString = start.Format(TestStartTimeFormat)

	samplesUsers = cmd.GenuineUsers(true)
	if _, ok := samplesUsers[userId]; userId > 0 && !ok {
		log.Fatal("No such user")
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
