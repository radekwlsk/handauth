package main

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/signature"
	"github.com/radekwlsk/handauth/signature/features"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"log"
	"math"
	"os"
	"path"
)

type Grid struct {
	Data *mat.Dense
}

func (g Grid) Min() float64       { return mat.Min(g.Data) }
func (g Grid) Max() float64       { return mat.Max(g.Data) }
func (g Grid) Dims() (c, r int)   { r, c = g.Data.Dims(); return c, r }
func (g Grid) Z(c, r int) float64 { return g.Data.At(r, c) }
func (g Grid) X(c int) float64 {
	_, n := g.Data.Dims()
	if c < 0 || c >= n {
		panic("column index out of range")
	}
	return float64(c)
}
func (g Grid) Y(r int) float64 {
	m, _ := g.Data.Dims()
	if r < 0 || r >= m {
		panic("row index out of range")
	}
	return float64(r)
}

type integerTicks struct{}

func (integerTicks) Ticks(min, max float64) []plot.Tick {
	var t []plot.Tick
	for i := math.Trunc(min); i <= max; i++ {
		t = append(t, plot.Tick{Value: i, Label: fmt.Sprint(i)})
	}
	return t
}

func plotHeatmap(data *mat.Dense, title string, filename string) {
	m := Grid{
		Data: data,
	}
	pal := palette.Heat(32, 1)
	h := plotter.NewHeatMap(m, pal)

	p, err := plot.New()
	if err != nil {
		log.Panic(err)
	}
	p.Title.Text = title
	p.X.Tick.Marker = integerTicks{}
	p.Y.Tick.Marker = integerTicks{}

	p.Add(h)

	// Create a legend.
	l, err := plot.NewLegend()
	if err != nil {
		log.Panic(err)
	}
	thumbs := plotter.PaletteThumbnailers(pal)
	for i := len(thumbs) - 1; i >= 0; i-- {
		t := thumbs[i]
		if i != 0 && i != len(thumbs)-1 {
			l.Add("", t)
			continue
		}
		var val float64
		switch i {
		case 0:
			val = h.Min
		case len(thumbs) - 1:
			val = h.Max
		}
		l.Add(fmt.Sprintf("%.2g", val), t)
	}

	p.X.Padding = 0
	p.Y.Padding = 0
	{
		r, c := data.Dims()
		p.X.Max, p.Y.Max = float64(c), float64(r)
	}

	img := vgimg.New(720, 480)
	dc := draw.New(img)

	l.Top = true
	// Calculate the width of the legend.
	r := l.Rectangle(dc)
	legendWidth := r.Max.X - r.Min.X
	l.YOffs = -p.Title.Font.Extents().Height // Adjust the legend down a little.

	l.Draw(dc)
	dc = draw.Crop(dc, 0, -legendWidth-vg.Millimeter, 0, 0) // Make space for the legend.
	p.Draw(dc)
	w, err := os.Create(path.Join("res", filename))
	if err != nil {
		log.Panic(err)
	}
	defer w.Close()
	png := vgimg.PngCanvas{Canvas: img}
	if _, err = png.WriteTo(w); err != nil {
		log.Panic(err)
	}
}

func newTrue() *bool {
	b := true
	return &b
}

func main() {
	flag.Parse()
	flags.AreaFilterOff = newTrue()
	flags.StdMeanFilterOff = newTrue()

	genuineSamplesUsers := cmd.GenuineUsers(false)
	users := map[uint8]*signature.UserModel{}
	{
		featuresChan := make(chan *signature.UserModel)

		for user, samples := range genuineSamplesUsers {
			split := int(math.Ceil(float64(len(samples)) * 0.2))
			go cmd.EnrollUserSync(uint8(user), samples[:split],
				uint16(*flags.Rows), uint16(*flags.Cols), featuresChan)
		}

		for range genuineSamplesUsers {
			f := <-featuresChan
			if f.Model != nil {
				users[f.Id] = f
				if *flags.VVerbose {
					log.Printf("\tEnrolled user %03d\n", f.Id)
				}
			}
		}
		close(featuresChan)
	}
	heatmapMatMap := map[features.FeatureType]*mat.Dense{
		features.LengthFeatureType:   mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.GradientFeatureType: mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.HOGFeatureType:      mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.CornersFeatureType:  mat.NewDense(*flags.Rows, *flags.Cols, nil),
	}
	for r := 0; r < *flags.Rows; r++ {
		for c := 0; c < *flags.Cols; c++ {
			for ftrType := range heatmapMatMap {
				vector := make([]float64, 0)
				for _, user := range users {
					ftr := user.Model.Grid(r, c)[ftrType]
					vector = append(vector, ftr.Value())
				}
				heatmapMatMap[ftrType].Set(r, c, stat.Variance(vector, nil))
			}
		}
	}

	for ftrType, heatmap := range heatmapMatMap {
		if mat.Min(heatmap) >= mat.Max(heatmap) {
			log.Println(fmt.Sprintf("empty matrix %s of values %f", ftrType, mat.Min(heatmap)))
			continue
		}
		plotHeatmap(heatmap, fmt.Sprintf("%s Variation Between Subjects", ftrType), ftrType.String()+".png")
	}
}
