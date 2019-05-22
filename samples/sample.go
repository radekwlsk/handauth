package samples

import (
	"fmt"
	"github.com/google/uuid"
	"gocv.io/x/gocv"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	TargetWidth = 500.0
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
		mat:    gocv.Mat{},
		height: 0,
		width:  0,
		ratio:  0.0,
	}
	err := s.read(filename)
	if s.Empty() {
		return nil, fmt.Errorf("could not read file %s", filename)
	}
	if Debug {
		logger.Printf("read sample from %s: %#v\n", filename, s)
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

func (sample *Sample) Area() int {
	return sample.mat.Total()
}

func (sample *Sample) read(name string) error {
	sample.mat = gocv.IMRead(name, gocv.IMReadGrayScale)
	if sample.mat.Empty() {
		return fmt.Errorf("failed to read sample: %s", name)
	}
	sample.Update()
	return nil
}

func (sample *Sample) Preprocess(ratio float64) {
	sample.Normalize()
	if Debug {
		sample.Save("res", "normalized", false)
	}
	sample.Foreground()
	if Debug {
		sample.Save("res", "foreground", false)
	}
	sample.Crop()
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
	//sample.toLines()
	//if Debug { sample.Save("res", "lines approximated", false) }
}

func (sample *Sample) Update() {
	sample.height = uint16(sample.mat.Rows())
	sample.width = uint16(sample.mat.Cols())
	sample.ratio = float64(sample.width) / float64(sample.height)
}

func (sample *Sample) Normalize() {
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
	sample.mat = dst
}

func (sample *Sample) Foreground() {
	defer sample.Update()
	dst := gocv.NewMat()

	gocv.GaussianBlur(sample.mat, &dst, image.Pt(3, 3), 0, 0, gocv.BorderReplicate)
	gocv.Threshold(dst, &dst, 0.0, 255.0, gocv.ThresholdBinaryInv+gocv.ThresholdOtsu)

	_ = sample.mat.Close()
	sample.mat = dst
}

func (sample *Sample) toLines() {
	matLines := gocv.NewMat()
	defer matLines.Close()
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
		gocv.Line(&dst, pt1, pt2, color.RGBA{R: 255, G: 255, B: 255}, 1)
	}

	_ = sample.mat.Close()
	sample.mat = dst
}

func (sample *Sample) zhangSuen() {
	defer sample.Update()
	dst := gocv.NewMat()

	ZhangSuen(sample.mat, &dst)

	_ = sample.mat.Close()
	sample.mat = dst
}

func (sample *Sample) Crop() {
	defer sample.Update()
	dst := gocv.NewMat()

	contours := gocv.FindContours(sample.mat, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	rect := gocv.BoundingRect(contours[0])
	for _, c := range contours[1:] {
		rect = rect.Union(gocv.BoundingRect(c))
	}
	dst = sample.mat.Region(rect)

	_ = sample.mat.Close()
	sample.mat = dst
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
	sample.mat = dst
}

func (sample *Sample) CenterOfMass() image.Point {
	m := gocv.Moments(sample.mat, true)
	x := int(math.Round(m["m10"] / m["m00"]))
	y := int(math.Round(m["m01"] / m["m00"]))
	return image.Point{X: x, Y: y}
}

func (sample *Sample) Save(dir, filename string, show bool) string {
	filename = strings.ReplaceAll(filename, " ", "_")
	filepath := fmt.Sprintf("%s-%s.png", path.Join(dir, filename), uuid.New().String()[:8])
	f, err := os.Create(filepath)
	if err != nil {
		log.Println(err)
	}
	img, err := sample.mat.ToImage()
	if err != nil {
		log.Println(err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		log.Println(err)
	}
	if err := f.Close(); err != nil {
		log.Println(err)
	}
	if Debug {
		log.Printf("saved sample %#v as %s", sample, f.Name())
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

func (sample *Sample) GoString() string {
	return fmt.Sprintf(
		"<%T %dx%d area %d>",
		sample,
		int(sample.width),
		int(sample.height),
		sample.Area(),
	)
}

func (sample *Sample) Close() {
	_ = sample.mat.Close()
}
