package samples

import (
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"math"
	"sync"
)

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

	region := sg.sample.mat.Region(rect2)
	mat := gocv.NewMat()
	region.CopyTo(&mat)

	sg.mutex.Unlock()

	s := &Sample{
		mat:    mat,
		height: sg.config.fieldHeight,
		width:  sg.config.fieldWidth,
		ratio:  float64(region.Cols()) / float64(region.Rows()),
	}
	return s
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
	w = uint16(math.Ceil((10 * width) / (float64(3*cols) + 7)))
	xs = uint16(math.Floor(0.3 * float64(w)))
	if float64(xs*(cols-1)) > width {
		panic(fmt.Sprintf("decrease number of columns or stride value"))
	}
	h = uint16(math.Ceil((10 * height) / (float64(3*rows) + 7)))
	ys = uint16(math.Floor(0.3 * float64(h)))
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

func (sg *SampleGrid) Save(dir, filename string, show bool) {
	for r := 0; r < int(sg.config.rows); r++ {
		sample := sg.At(r, -1)
		sample.Save(dir, fmt.Sprintf("%s-row%d", filename, r), show)
		sample.Close()
		for c := 0; c < int(sg.config.cols); c++ {
			sample := sg.At(r, c)
			sample.Save(dir, fmt.Sprintf("%s-col%d_row%d", filename, c, r), show)
			sample.Close()
		}
	}
	for c := 0; c < int(sg.config.cols); c++ {
		sample := sg.At(-1, c)
		sample.Save(dir, fmt.Sprintf("%s-col%d", filename, c), show)
		sample.Close()
	}
}
