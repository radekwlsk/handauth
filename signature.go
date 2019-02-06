package handauth

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

func (im *Image) read(name string) {
	mat := gocv.IMRead(name, gocv.IMReadGrayScale)
	im.mat = mat
	if im.mat.Empty() {
		fmt.Printf("Failed to read image: %s\n", name)
		os.Exit(1)
	}
	im.update()
}

func (im *Image) preprocess() {
	im.normalize()
	im.foreground()
	im.crop()
	im.resize(400)
}

func (im *Image) update() {
	im.height = im.mat.Rows()
	im.width = im.mat.Cols()
	im.ratio = float64(im.width) / float64(im.height)
}

func (im *Image) normalize() {
	defer im.update()
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
	defer im.update()
	dst := gocv.NewMat()

	gocv.Threshold(im.mat, &dst, 0.0, 255.0, gocv.ThresholdBinaryInv+gocv.ThresholdOtsu)

	im.mat = dst
}

func (im *Image) crop() {
	defer im.update()
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
	defer im.update()
	dst := gocv.NewMat()

	point := image.Point{
		X: width,
		Y: int(float64(width) / im.ratio),
	}
	gocv.Resize(im.mat, &dst, point, 0.0, 0.0, gocv.InterpolationNearestNeighbor)

	im.mat = dst
}

type Signature struct {
	user  string
	image Image
}

func (s *Signature) Image() Image {
	return s.image
}

func NewSignature(username, filename string) Signature {
	im := Image{
		mat:    gocv.NewMat(),
		height: 0,
		width:  0,
		ratio:  0.0,
	}
	im.read(filename)
	s := Signature{
		user:  username,
		image: im,
	}
	return s
}

func (s *Signature) Preprocess() {
	s.image.preprocess()
}

func (s *Signature) Show() {
	window := gocv.NewWindow(fmt.Sprintf("%s's signature (%dx%d)", s.user, s.image.width, s.image.height))
	defer window.Close()
	window.ResizeWindow(s.image.width, s.image.height)
	window.IMShow(s.image.mat)
	for window.IsOpen() {
		if window.WaitKey(1) > 0 {
			break
		}
	}
}
