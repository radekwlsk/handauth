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

func (sample *Sample) Load(mat gocv.Mat) {
	sample.mat = mat
	sample.Update()
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
	sample.ZhangSuen()
	if Debug {
		sample.Save("res", "thinned", false)
	}
	//sample.ToLines()
	//if Debug { sample.Save("res", "lines", false) }
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

func pointDist(pt1, pt2 image.Point) float64 {
	return math.Sqrt(
		math.Pow(float64(pt1.X-pt2.X), 2) + math.Pow(float64(pt1.Y-pt2.Y), 2),
	)
}

func (sample *Sample) ToLines() {
	matLines := gocv.NewMat()
	defer matLines.Close()
	dst := gocv.NewMatWithSize(sample.Height(), sample.Width(), 0)
	gocv.GaussianBlur(sample.mat, &dst, image.Pt(3, 3), 0, 0, gocv.BorderReplicate)

	gocv.HoughLinesPWithParams(
		dst,
		&matLines,
		45,
		2*math.Pi/180,
		50,
		15,
		7,
	)

	dst.Close()
	dst = gocv.NewMatWithSize(sample.Height(), sample.Width(), 0)
	bounds := make([][2]image.Point, 0)
	for i := 0; i < matLines.Rows(); i++ {
		pt1 := image.Pt(int(matLines.GetVeciAt(i, 0)[0]), int(matLines.GetVeciAt(i, 0)[1]))
		pt2 := image.Pt(int(matLines.GetVeciAt(i, 0)[2]), int(matLines.GetVeciAt(i, 0)[3]))
		bounds = append(bounds, [2]image.Point{pt1, pt2})
	}
Outer:
	for i, pts := range bounds {
		for j, pts2 := range bounds {
			if i == j {
				continue
			}
			rect := image.Rect(pts[0].X, pts[0].Y, pts[1].X, pts[1].Y)
			rect2 := image.Rect(pts2[0].X, pts2[0].Y, pts2[1].X, pts2[1].Y)
			//
			insideOther := rect.In(rect2)
			nearOther := (pointDist(pts[0], pts2[0]) < 11 && pointDist(pts[1], pts2[1]) < 21) ||
				(pointDist(pts[0], pts2[0]) < 21 && pointDist(pts[1], pts2[1]) < 11)
			nearOther2 := (pointDist(pts[0], pts2[0]) < 35 && pointDist(pts[1], pts2[1]) < 35) ||
				(pointDist(pts[0], pts2[0]) < 35 && pointDist(pts[1], pts2[1]) < 35)
			isShorter := pointDist(pts[0], pts[1]) < pointDist(pts2[0], pts2[1])
			if (nearOther2 && insideOther) || (nearOther && isShorter) {
				continue Outer
			}
		}
		//pt1 := image.Pt(int(matLines.GetVeciAt(i, 0)[0]), int(matLines.GetVeciAt(i, 0)[1]))
		//pt2 := image.Pt(int(matLines.GetVeciAt(i, 0)[2]), int(matLines.GetVeciAt(i, 0)[3]))
		gocv.Line(&dst, pts[0], pts[1], color.RGBA{R: 255, G: 255, B: 255}, 1)
	}

	_ = sample.mat.Close()
	sample.mat = dst
}

func (sample *Sample) ZhangSuen() {
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
	if rect.Empty() {
		log.Fatal("cropping yields empty matrix")
	}

	region := sample.mat.Region(rect)
	region.CopyTo(&dst)

	_ = region.Close()
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


func (sample *Sample) Enlarge(width int, height int, c *color.RGBA) {
	defer sample.Update()
	dst := gocv.NewMat()
	
	ml := 0
	mr := 0
	mt := 0
	mb := 0
	if hgt := height - sample.Height(); hgt > 0 {
		h := int(math.Floor(float64(hgt) / 2))
		r := hgt - (2*h)
		mt = h
		mb = h + r
	}
	if wdt := width - sample.Width(); wdt > 0 {
		w := int(math.Floor(float64(wdt) / 2))
		r := wdt - (2*w)
		ml = w
		mr = w + r
	}
	
	if c == nil {
		c = &color.RGBA{A: 255}
	}
	
	gocv.CopyMakeBorder(sample.mat, &dst, mt, mb, ml, mr, gocv.BorderConstant, *c)
	
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
	filename = fmt.Sprintf("%s-%s", filename, uuid.New().String()[:8])
	if !strings.HasSuffix(filename, ".png") {
		filename = fmt.Sprintf("%s.png", filename)
	}
	filepath := path.Join(dir, filename)
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

type histXYer struct {
	vals []float64
}

func (h histXYer) Increment(i int) {
	h.vals[i] += 1
}

func (h histXYer) Len() int {
	return len(h.vals)
}

func (h histXYer) XY(i int) (x, y float64) {
	return float64(i), h.vals[i]
}

func (sample *Sample) Histogram(dir, filename string, bins int) error {
	vals := histXYer{make([]float64, 256)}
	matVals := sample.mat.DataPtrUint8()
	for _, v := range matVals {
		vals.Increment(int(v))
	}

	p, err := plot.New()
	if err != nil {
		return err
	}
	p.Title.Text = "Histogram"
	h, err := plotter.NewHistogram(vals, bins)
	if err != nil {
		return err
	}
	h.Normalize(1)
	p.Add(h)
	if !strings.HasSuffix(filename, ".png") && !strings.HasSuffix(filename, ".jpg") {
		filename += ".png"
	}
	if err = p.Save(480, 320, path.Join(dir, filename)); err != nil {
		return err
	}
	return nil
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
