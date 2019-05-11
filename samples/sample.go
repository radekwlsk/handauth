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
	TargetWidth = 500.0
	Stride      = 7.0
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

func (sample *Sample) Area() int {
	return sample.mat.Total()
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
	//sample.toLines()
	//if Debug { sample.Save("res", "lines approximated", false) }
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

func (sample *Sample) GoString() string {
	return fmt.Sprintf(
		"<Sample %dx%d area %d>",
		int(sample.width),
		int(sample.height),
		sample.Area(),
	)
}

func (sample *Sample) Close() {
	_ = sample.mat.Close()
}

type GridConfig struct {
	height      uint16
	width       uint16
	fieldHeight uint16
	fieldWidth  uint16
	rowHeight   uint16
	colWidth    uint16
	yStride     uint16
	xStride     uint16
	rows        uint16
	cols        uint16
}

func (gc GridConfig) GoString() string {
	return fmt.Sprintf("<%T %dx%d f:%dx%d s:%dx%d r:%d(%d) c:%d(%d)>", gc, gc.width, gc.height,
		gc.fieldWidth, gc.fieldHeight, gc.xStride, gc.yStride, gc.rows, gc.rowHeight, gc.cols, gc.colWidth)
}

func calcOverlappingGridSize(height, width float64, rows, cols uint16) (h, w, ys, xs uint16) {
	w = uint16(math.Floor((10 * width) / (float64(3*cols) + 7)))
	//xs = uint16(Stride)
	xs = uint16(math.Floor(0.3 * float64(w)))
	if float64(xs*(cols-1)) > width {
		panic(fmt.Sprintf("decrease number of columns or stride value"))
	}
	//w = uint16(math.Ceil(width - (float64(xs)*float64(cols-1))))
	//ys = uint16(math.Ceil(float64(xs) * float64(height)/float64(width)))
	//h = uint16(math.Ceil(float64(w)*float64(height)/float64(width)))
	//ys = uint16(math.Ceil((height-() / float64(rows-1)))
	//ys = uint16(math.Floor((height-float64(h)) / float64(rows-1)))
	//ys = uint16(math.Floor(0.3 * float64(h)))
	h = uint16(math.Floor((10 * height) / (float64(3*rows) + 7)))
	ys = uint16(math.Floor(0.3 * float64(h)))
	//if float64(ys * (rows-1)) > height {
	//	panic(fmt.Sprintf("decrease number of rows or stride value"))
	//}
	//h = uint16(math.Ceil(height - float64(ys)*float64(rows-1)))
	if h <= 5 {
		panic(fmt.Sprintf("decrease number of rows (%.0f, %.0f)", height, width))
	}
	if w <= 5 {
		panic(fmt.Sprintf("decrease number of columns (%.0f, %.0f)", height, width))
	}
	return h, w, ys, xs
}

func calcGridSize(height, width float64, rows, cols uint16) (h, w uint16) {
	h = uint16(math.Floor(float64(height) / float64(rows)))
	w = uint16(math.Floor(float64(width) / float64(cols)))
	if h <= 3 {
		panic(fmt.Sprintf("decrease number of rows (%.0f, %.0f)", height, width))
	}
	if w <= 3 {
		panic(fmt.Sprintf("decrease number of columns (%.0f, %.0f)", height, width))
	}
	return h, w
}

func NewGridConfig(sample *Sample, rows, cols uint16) GridConfig {
	gh, gw, ys, xs := calcOverlappingGridSize(float64(sample.height), float64(sample.width), rows, cols)
	rh, cw := calcGridSize(float64(sample.height), float64(sample.width), rows, cols)
	return GridConfig{
		height:      sample.height,
		width:       sample.width,
		fieldHeight: gh,
		fieldWidth:  gw,
		rowHeight:   rh,
		colWidth:    cw,
		xStride:     xs,
		yStride:     ys,
		rows:        rows,
		cols:        cols,
	}
}

func (gc *GridConfig) FieldArea() float64 {
	return float64(gc.fieldWidth * gc.fieldHeight)
}

func (gc *GridConfig) RowArea() float64 {
	return float64(gc.width * gc.rowHeight)
}

func (gc *GridConfig) ColArea() float64 {
	return float64(gc.height * gc.colWidth)
}

func (gc *GridConfig) FieldRect(row, col int) image.Rectangle {
	x0 := col * int(gc.xStride)
	x1 := x0 + int(gc.fieldWidth)
	if x1 > int(gc.width) || col == int(gc.cols-1) {
		x1 = int(gc.width)
	}
	y0 := row * int(gc.yStride)
	y1 := y0 + int(gc.fieldHeight)
	if y1 > int(gc.height) || row == int(gc.rows-1) {
		y1 = int(gc.height)
	}
	return image.Rect(x0, y0, x1, y1)
}

func (gc *GridConfig) RowRect(row int) image.Rectangle {
	x0 := 0
	x1 := int(gc.width)
	y0 := row * int(gc.rowHeight)
	y1 := y0 + int(gc.rowHeight)
	if y1 > int(gc.height) || row == int(gc.rows-1) {
		y1 = int(gc.height)
	}
	return image.Rect(x0, y0, x1, y1)
}

func (gc *GridConfig) ColRect(col int) image.Rectangle {
	x0 := col * int(gc.colWidth)
	x1 := x0 + int(gc.colWidth)
	if x1 > int(gc.width) || col == int(gc.cols-1) {
		x1 = int(gc.width)
	}
	y0 := 0
	y1 := int(gc.height)
	return image.Rect(x0, y0, x1, y1)
}

type SampleGrid struct {
	sample *Sample
	config GridConfig
	mutex  sync.Mutex
}

func (sg *SampleGrid) Config() GridConfig {
	return sg.config
}

func NewSampleGrid(sample *Sample, rows, cols uint16) *SampleGrid {
	return &SampleGrid{
		sample: sample,
		config: NewGridConfig(sample, rows, cols),
	}
}

func (sg SampleGrid) GoString() string {
	return fmt.Sprintf("<%T %#v>", sg, sg.config)
}

func (sg *SampleGrid) At(row, col int) *Sample {
	var rect image.Rectangle
	if col < 0 {
		rect = sg.config.RowRect(row)
	} else if row < 0 {
		rect = sg.config.ColRect(col)
	} else {
		rect = sg.config.FieldRect(row, col)
	}

	rect2 := rect.Intersect(image.Rect(0, 0, int(sg.config.width), int(sg.config.height)))
	if rect.Empty() {
		panic(fmt.Errorf("empty rect at (%d, %d):\n%v\n%v\nconfig: %#v", row, col, rect, rect2, sg.config))
	}

	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	region := sg.sample.mat.Region(rect2)
	s := &Sample{
		mat:    region,
		height: sg.config.fieldHeight,
		width:  sg.config.fieldWidth,
		ratio:  float64(region.Cols()) / float64(region.Rows()),
	}
	return s
}

func (sg *SampleGrid) Save(dir, filename string, show bool) {
	for r := 0; r < int(sg.config.rows); r++ {
		sample := sg.At(r, -1)
		sample.Save(dir, fmt.Sprintf("%s-row%d", filename, r), show)
		for c := 0; c < int(sg.config.cols); c++ {
			sample := sg.At(r, c)
			sample.Save(dir, fmt.Sprintf("%s-col%d_row%d", filename, c, r), show)
		}
	}
	for c := 0; c < int(sg.config.cols); c++ {
		sample := sg.At(-1, c)
		sample.Save(dir, fmt.Sprintf("%s-col%d", filename, c), show)
	}
}
