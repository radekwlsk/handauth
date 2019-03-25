package auth

import (
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"math"
	"os"
)

type Image struct {
	mat    gocv.Mat
	height int
	width  int
	ratio  float64
}

func NewImage(filename string) *Image {
	im := Image{
		mat:    gocv.NewMat(),
		height: 0,
		width:  0,
		ratio:  0.0,
	}
	im.read(filename)
	return &im
}

func (im *Image) Mat() gocv.Mat {
	return im.mat
}

func (im *Image) Height() int {
	return im.height
}

func (im *Image) Width() int {
	return im.width
}

func (im *Image) Ratio() float64 {
	return im.ratio
}

func (im *Image) read(name string) {
	mat := gocv.IMRead(name, gocv.IMReadGrayScale)
	im.mat = mat
	if im.mat.Empty() {
		fmt.Printf("Failed to read image: %s\n", name)
		os.Exit(1)
	}
	im.Update()
}

func (im *Image) Preprocess() {
	im.normalize()
	im.foreground()
	im.crop()
	im.resize(400)
}

func (im *Image) Update() {
	im.height = im.mat.Rows()
	im.width = im.mat.Cols()
	im.ratio = float64(im.width) / float64(im.height)
}

func (im *Image) normalize() {
	defer im.Update()
	dst := gocv.NewMat()

	gocv.BilateralFilter(im.mat, &dst, 5, 75, 75)
	gocv.Normalize(im.mat, &dst, 0, 255, gocv.NormMinMax)
	gocv.ConvertScaleAbs(im.mat, &dst, 1.1, 20)
	lookup := gocv.NewMatWithSize(1, 256, gocv.MatTypeCV8U)
	for i := 0; i < lookup.Cols(); i++ {
		val := uint8(math.Max(0, math.Min(255, math.Pow(float64(i)/255.0, 2)*255.0)))
		lookup.SetUCharAt(0, i, val)
	}
	gocv.LUT(im.mat, lookup, &dst)

	im.mat = dst
}

func (im *Image) foreground() {
	defer im.Update()
	dst := gocv.NewMat()

	gocv.Threshold(im.mat, &dst, 0.0, 255.0, gocv.ThresholdBinaryInv+gocv.ThresholdOtsu)

	im.mat = dst
}

func (im *Image) crop() {
	defer im.Update()
	dst := gocv.NewMat()

	contours := gocv.FindContours(im.mat, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	rect := gocv.BoundingRect(contours[0])
	for _, c := range contours[1:] {
		rect = rect.Union(gocv.BoundingRect(c))
	}
	dst = im.mat.Region(rect)

	im.mat = dst.Clone()
}

func (im *Image) resize(width int) {
	defer im.Update()
	dst := gocv.NewMat()

	point := image.Point{
		X: width,
		Y: int(float64(width) / im.ratio),
	}
	gocv.Resize(im.mat, &dst, point, 0.0, 0.0, gocv.InterpolationNearestNeighbor)

	im.mat = dst
}
