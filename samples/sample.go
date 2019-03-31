package samples

import (
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"math"
	"os"
)

type Sample struct {
	mat    gocv.Mat
	height int
	width  int
	ratio  float64
}

func NewSample(filename string) *Sample {
	s := &Sample{
		mat:    gocv.NewMat(),
		height: 0,
		width:  0,
		ratio:  0.0,
	}
	s.read(filename)
	return s
}

func (sample *Sample) Mat() gocv.Mat {
	return sample.mat
}

func (sample *Sample) Height() int {
	return sample.height
}

func (sample *Sample) Width() int {
	return sample.width
}

func (sample *Sample) Ratio() float64 {
	return sample.ratio
}

func (sample *Sample) read(name string) {
	mat := gocv.IMRead(name, gocv.IMReadGrayScale)
	sample.mat = mat
	if sample.mat.Empty() {
		fmt.Printf("Failed to read sample: %s\n", name)
		os.Exit(1)
	}
	sample.Update()
}

func (sample *Sample) Preprocess(ratio float64) {
	sample.normalize()
	sample.foreground()
	sample.crop()
	sample.resize(400, ratio)
}

func (sample *Sample) Update() {
	sample.height = sample.mat.Rows()
	sample.width = sample.mat.Cols()
	sample.ratio = float64(sample.width) / float64(sample.height)
}

func (sample *Sample) normalize() {
	defer sample.Update()
	dst := gocv.NewMat()

	gocv.BilateralFilter(sample.mat, &dst, 5, 75, 75)
	gocv.Normalize(sample.mat, &dst, 0, 255, gocv.NormMinMax)
	gocv.ConvertScaleAbs(sample.mat, &dst, 1.1, 20)
	lookup := gocv.NewMatWithSize(1, 256, gocv.MatTypeCV8U)
	for i := 0; i < lookup.Cols(); i++ {
		val := uint8(math.Max(0, math.Min(255, math.Pow(float64(i)/255.0, 2)*255.0)))
		lookup.SetUCharAt(0, i, val)
	}
	gocv.LUT(sample.mat, lookup, &dst)
	sample.mat = dst
}

func (sample *Sample) foreground() {
	defer sample.Update()
	dst := gocv.NewMat()

	gocv.Threshold(sample.mat, &dst, 0.0, 255.0, gocv.ThresholdBinaryInv+gocv.ThresholdOtsu)

	sample.mat = dst
}

func (sample *Sample) crop() {
	defer sample.Update()
	dst := gocv.NewMat()

	contours := gocv.FindContours(sample.mat, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	rect := gocv.BoundingRect(contours[0])
	for _, c := range contours[1:] {
		rect = rect.Union(gocv.BoundingRect(c))
	}
	dst = sample.mat.Region(rect)

	sample.mat = dst.Clone()
}

func (sample *Sample) resize(width int, ratio float64) {
	defer sample.Update()
	dst := gocv.NewMat()

	if ratio == 0.0 {
		ratio = sample.ratio
	}

	point := image.Point{
		X: width,
		Y: int(float64(width) / ratio),
	}
	gocv.Resize(sample.mat, &dst, point, 0.0, 0.0, gocv.InterpolationNearestNeighbor)

	sample.mat = dst
}

func (sample *Sample) Show() {
	window := gocv.NewWindow(fmt.Sprintf("(%dx%d)", sample.Width(), sample.Height()))
	defer window.Close()
	window.ResizeWindow(sample.Width(), sample.Height())
	window.IMShow(sample.Mat())
	for window.IsOpen() {
		if window.WaitKey(1) > 0 {
			break
		}
	}
}

func (sample *Sample) String() string {
	return fmt.Sprintf(
		"<Sample %dx%d (%.2f)>",
		int(sample.width),
		int(sample.height),
		sample.ratio)
}

type SampleGrid struct {
	sample      *Sample
	fieldHeight int
	fieldWidth  int
	stride      int
	rows        int
	cols        int
	fields      [][]*Sample
}

func calcGridDim(width, expectedSize, stride float64) (dim int, size float64) {
	n := math.Round(((width - expectedSize) / stride) + 1)
	return int(n), math.Ceil(width - stride*(n-1))
}

func NewSampleGrid(sample *Sample, size, stride float64) *SampleGrid {
	rows, height := calcGridDim(float64(sample.height), size, stride)
	cols, width := calcGridDim(float64(sample.width), size, stride)
	fields := make([][]*Sample, rows)
	for i := range fields {
		fields[i] = make([]*Sample, cols)
		for j := range fields[i] {
			fields[i][j] = nil
		}
	}
	return &SampleGrid{
		sample:      sample,
		fieldHeight: int(height),
		fieldWidth:  int(width),
		stride:      int(stride),
		rows:        rows,
		cols:        cols,
		fields:      fields,
	}
}

func (sgrid *SampleGrid) String() string {
	return fmt.Sprintf(
		"<SampleGrid %dx%d, (%d, %d), %d>",
		sgrid.cols,
		sgrid.rows,
		sgrid.fieldWidth,
		sgrid.fieldHeight,
		sgrid.stride)
}

func (sgrid *SampleGrid) At(row, col int) *Sample {
	if sgrid.fields[row][col] == nil {
		s := &Sample{
			mat:    gocv.NewMat(),
			height: sgrid.fieldHeight,
			width:  sgrid.fieldWidth,
			ratio:  float64(sgrid.fieldWidth) / float64(sgrid.fieldHeight),
		}
		x1 := col * sgrid.stride
		y1 := row * sgrid.stride
		x2 := x1 + sgrid.fieldWidth
		if x2 > sgrid.sample.width {
			x2 = sgrid.sample.width
		}
		y2 := y1 + sgrid.fieldHeight
		if y2 > sgrid.sample.height {
			y2 = sgrid.sample.height
		}
		rect := image.Rect(x1, y1, x2, y2)
		s.mat = sgrid.sample.mat.Region(rect)
		sgrid.fields[row][col] = s
	}
	return sgrid.fields[row][col]
}

func (sgrid *SampleGrid) Show() {
	var windows []*gocv.Window
	for i := range sgrid.fields {
		for j := range sgrid.fields[i] {
			sample := sgrid.At(i, j)
			window := gocv.NewWindow(fmt.Sprintf("[%d, %d]", i, j))
			window.ResizeWindow(sample.Width(), sample.Height())
			window.IMShow(sample.Mat())
			windows = append(windows, window)
		}
	}
loop:
	for {
		for _, window := range windows {
			if window.IsOpen() {
				if window.WaitKey(1) > 0 {
					break loop
				}
			}
		}
	}
	for _, window := range windows {
		if window.IsOpen() {
			window.Close()
		}
	}
}
