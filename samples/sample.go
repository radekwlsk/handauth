package samples

import (
	"fmt"
	"github.com/google/uuid"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

const (
	TargetWidth = 400.0
	Stride      = 20.0
)

var Debug = false
var logger = log.New(os.Stdout, "[sample] ", log.Lshortfile+log.Ltime)

type Sample struct {
	mat    gocv.Mat
	height uint16
	width  uint16
	ratio  float64
}

func NewSample(filename string) (*Sample, error) {
	s := &Sample{
		mat:    gocv.NewMat(),
		height: 0,
		width:  0,
		ratio:  0.0,
	}
	err := s.read(filename)
	if s.Empty() {
		return nil, fmt.Errorf("could not read file %s", filename)
	}
	if Debug {
		logger.Printf("read sample from %s: %s\n", filename, s)
	}
	return s, err
}

func (sample *Sample) Copy() *Sample {
	return &Sample{
		mat:    sample.mat.Clone(),
		height: sample.height,
		width:  sample.width,
		ratio:  sample.ratio,
	}
}

func (sample *Sample) Empty() bool {
	return sample.mat.Empty()
}

func (sample *Sample) MatType() gocv.MatType {
	return sample.mat.Type()
}

func (sample *Sample) Mat() gocv.Mat {
	return sample.mat
}

func (sample *Sample) Height() int {
	return int(sample.height)
}

func (sample *Sample) Width() int {
	return int(sample.width)
}

func (sample *Sample) Ratio() float64 {
	return sample.ratio
}

func (sample *Sample) read(name string) error {
	mat := gocv.IMRead(name, gocv.IMReadGrayScale)
	sample.mat = mat
	if sample.mat.Empty() {
		return fmt.Errorf("failed to read sample: %s", name)
	}
	sample.Update()
	return nil
}

func (sample *Sample) Preprocess(ratio float64) {
	sample.normalize()
	if Debug {
		sample.Save("res", "normalized", false)
	}
	sample.foreground()
	if Debug {
		sample.Save("res", "foreground", false)
	}
	sample.crop()
	if Debug {
		sample.Save("res", "cropped", false)
	}
	sample.Resize(TargetWidth, ratio)
	if Debug {
		sample.Save("res", "resized", false)
	}
	sample.zhangSuen()
	if Debug {
		sample.Save("res", "thinned", false)
	}
	sample.toLines()
	if Debug {
		sample.Save("res", "lines approximated", false)
	}
}

func (sample *Sample) Update() {
	sample.height = uint16(sample.mat.Rows())
	sample.width = uint16(sample.mat.Cols())
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

	_ = sample.mat.Close()
	sample.mat = dst.Clone()
	_ = dst.Close()
}

func (sample *Sample) foreground() {
	defer sample.Update()
	dst := gocv.NewMat()

	gocv.Threshold(sample.mat, &dst, 0.0, 255.0, gocv.ThresholdBinaryInv+gocv.ThresholdOtsu)

	_ = sample.mat.Close()
	sample.mat = dst.Clone()
	_ = dst.Close()
}

func (sample *Sample) toLines() {
	matLines := gocv.NewMat()
	dst := gocv.NewMatWithSize(sample.Height(), sample.Width(), 0)

	gocv.HoughLinesPWithParams(
		sample.mat,
		&matLines,
		1,
		math.Pi/180,
		17,
		3,
		11,
	)

	for i := 0; i < matLines.Rows(); i++ {
		pt1 := image.Pt(int(matLines.GetVeciAt(i, 0)[0]), int(matLines.GetVeciAt(i, 0)[1]))
		pt2 := image.Pt(int(matLines.GetVeciAt(i, 0)[2]), int(matLines.GetVeciAt(i, 0)[3]))
		gocv.Line(&dst, pt1, pt2, color.RGBA{255, 255, 255, 0}, 1)
	}

	_ = matLines.Close()
	_ = sample.mat.Close()
	sample.mat = dst.Clone()
	_ = dst.Close()
}

func (sample *Sample) zhangSuen() {
	s1Flag := true
	s2Flag := true
	for s1Flag || s2Flag {
		s1Marks := make([][2]int, 0)
		for r := 0; r < sample.Height(); r++ {
			for c := 0; c < sample.Width(); c++ {
				if Step1ConditionsMet(sample, r, c) {
					s1Marks = append(s1Marks, [2]int{r, c})
				}
			}
		}
		s1Flag = len(s1Marks) > 0
		if s1Flag {
			for _, rc := range s1Marks {
				sample.mat.SetUCharAt(rc[0], rc[1], 0)
			}
		}

		s2Marks := make([][2]int, 0)
		for r := 0; r < sample.Height(); r++ {
			for c := 0; c < sample.Width(); c++ {
				if Step2ConditionsMet(sample, r, c) {
					s2Marks = append(s2Marks, [2]int{r, c})
				}
			}
		}
		s2Flag = len(s2Marks) > 0
		if s2Flag {
			for _, rc := range s2Marks {
				sample.mat.SetUCharAt(rc[0], rc[1], 0)
			}
		}
	}
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

	_ = sample.mat.Close()
	sample.mat = dst.Clone()
	_ = dst.Close()
}

func (sample *Sample) Resize(width int, ratio float64) {
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

	_ = sample.mat.Close()
	sample.mat = dst.Clone()
	_ = dst.Close()
}

func (sample *Sample) ColorModel() color.Model {
	switch sample.MatType() {
	case gocv.MatTypeCV8UC3:
		return color.RGBAModel
	default:
		return color.GrayModel
	}
}

func (sample *Sample) Bounds() image.Rectangle {
	return image.Rect(0, 0, sample.mat.Cols()-1, sample.mat.Rows()-1)
}

func (sample *Sample) At(x, y int) color.Color {
	c := sample.mat.GetUCharAt(y, x)
	return color.RGBA{c, c, c, 0}
}

func (sample *Sample) Save(dir, filename string, show bool) string {
	filename = strings.ReplaceAll(filename, " ", "_")
	filepath := fmt.Sprintf("%s-%s.png", path.Join(dir, filename), uuid.New().String()[:8])
	f, err := os.Create(filepath)
	if err != nil {
		log.Println(err)
	}
	if err := png.Encode(f, sample); err != nil {
		_ = f.Close()
		log.Println(err)
	}
	if err := f.Close(); err != nil {
		log.Println(err)
	}
	if Debug {
		log.Printf("saved sample %s as %s", sample, f.Name())
	}
	if show {
		command := "display"
		cmd := exec.Command(command, filepath)
		go func() {
			err := cmd.Run()
			if err != nil {
				log.Fatal(err)

			}
		}()
	}
	return filepath
}

func (sample *Sample) String() string {
	return fmt.Sprintf(
		"<Sample %dx%d (%.2f)>",
		int(sample.width),
		int(sample.height),
		sample.ratio)
}

func (sample *Sample) Close() {
	_ = sample.mat.Close()
}

type SampleGrid struct {
	sample      *Sample
	fieldHeight uint16
	fieldWidth  uint16
	yStride     uint8
	xStride     uint8
	rows        uint8
	cols        uint8
	mutex       sync.Mutex
}

//func calcGridDim(h, w float64) (rows, cols int) {
//	rows = int(math.Round(((h - ColWidth) / Stride) + 1))
//	cols = int(math.Round(((w - ColWidth) / Stride) + 1))
//	return
//}

func calcGridSize(height, width float64, rows, cols, yStride, xStride uint8) (uint16, uint16) {
	h := math.Ceil(height - float64(yStride)*float64(rows-1))
	w := math.Ceil(width - float64(xStride)*float64(cols-1))
	if h < 0 {
		panic(fmt.Sprintf("decrease number of rows (%.0f, %.0f)", height, width))
	}
	if w < 0 {
		panic(fmt.Sprintf("decrease number of columns (%.0f, %.0f)", height, width))
	}
	return uint16(h), uint16(w)
}

func NewSampleGrid(sample *Sample, rows, cols uint8) *SampleGrid {
	//yStride := uint8(math.Ceil(float64(sample.height) / float64(rows * 2)))
	//xStride := uint8(math.Ceil(float64(sample.width) / float64(cols * 2)))
	xStride := uint8(Stride)
	yStride := uint8(math.Floor(Stride / sample.ratio))

	height, width := calcGridSize(float64(sample.height), float64(sample.width), rows, cols, yStride, xStride)
	return &SampleGrid{
		sample:      sample,
		fieldHeight: height,
		fieldWidth:  width,
		yStride:     yStride,
		xStride:     xStride,
		rows:        rows,
		cols:        cols,
	}
}

func (sg *SampleGrid) String() string {
	return fmt.Sprintf(
		"<SampleGrid %dx%d, (%d, %d), [%d/%d]>",
		sg.rows,
		sg.cols,
		sg.fieldHeight,
		sg.fieldWidth,
		sg.yStride,
		sg.xStride,
	)
}

func (sg *SampleGrid) At(row, col int) *Sample {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()
	s := &Sample{
		mat:    gocv.NewMat(),
		height: sg.fieldHeight,
		width:  sg.fieldWidth,
		ratio:  float64(sg.fieldWidth) / float64(sg.fieldHeight),
	}
	var x1, y1, x2, y2 int
	if col >= 0 {
		x1 = col * int(sg.xStride)
	} else {
		x1 = 0
	}
	if row >= 0 {
		y1 = row * int(sg.yStride)
	} else {
		y1 = 0
	}
	x2 = x1 + int(sg.fieldWidth)
	if x2 > int(sg.sample.width) || col < 0 {
		x2 = int(sg.sample.width)
	}
	y2 = y1 + int(sg.fieldHeight)
	if y2 > int(sg.sample.height) || row < 0 {
		y2 = int(sg.sample.height)
	}

	if x2 <= x1 || y2 <= y1 {
		panic("at the disco")
	}

	rect := image.Rect(x1, y1, x2, y2)
	s.mat = sg.sample.mat.Region(rect)
	return s
}

func (sg *SampleGrid) Save(dir, filename string, show bool) {
	for r := 0; r < int(sg.rows); r++ {
		for c := 0; c < int(sg.cols); c++ {
			sample := sg.At(r, c)
			sample.Save(dir, fmt.Sprintf("%s-col%d_row%d", filename, c, r), show)
		}
	}
}
