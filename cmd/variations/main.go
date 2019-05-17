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
	"gonum.org/v1/plot/palette/brewer"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"log"
	"math"
	"os"
	"path"
	"time"
)

const TestStartTimeFormat = "20060201-150405"

var (
	fullResources bool
	addTimestamp  bool
	userId        int
	start         time.Time
	startString   string
	samplesUsers  map[int][]int
)

type Grid struct {
	Data *mat.Dense
}

func (g Grid) Min() float64     { return mat.Min(g.Data) }
func (g Grid) Max() float64     { return mat.Max(g.Data) }
func (g Grid) Dims() (c, r int) { r, c = g.Data.Dims(); return c, r }
func (g Grid) Z(c, r int) float64 {
	return g.Data.At(*flags.Rows-r-1, c)
}
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

func plotHeatmap(data *mat.Dense, title string) (*vgimg.Canvas, error) {
	m := Grid{
		Data: data,
	}
	pal, err := brewer.GetPalette(brewer.TypeSequential, "BuPu", 9)
	if err != nil {
		log.Fatal(err)
	}
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
		return nil, err
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

	p.X.Padding = 1
	p.Y.Padding = 1
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

	return img, nil
}

func savePlot(canvas *vgimg.Canvas, dir, filename string) error {
	w, err := os.Create(path.Join(dir, filename))
	if err != nil {
		return err
	}
	defer w.Close()
	png := vgimg.PngCanvas{Canvas: canvas}
	if _, err = png.WriteTo(w); err != nil {
		return err
	}
	return nil
}

func newTrue() *bool {
	b := true
	return &b
}

func variationBetweenUsers() map[features.FeatureType]*mat.Dense {
	heatMaps := map[features.FeatureType]*mat.Dense{
		features.LengthFeatureType:   mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.GradientFeatureType: mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.HOGFeatureType:      mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.CornersFeatureType:  mat.NewDense(*flags.Rows, *flags.Cols, nil),
	}

	users := map[uint8]*signature.UserModel{}
	{
		featuresChan := make(chan *signature.UserModel)

		for user, samples := range samplesUsers {
			go cmd.EnrollUserSync(uint8(user), samples, uint16(*flags.Rows), uint16(*flags.Cols), featuresChan)
		}

		for range samplesUsers {
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
	for r := 0; r < *flags.Rows; r++ {
		for c := 0; c < *flags.Cols; c++ {
			for ftrType := range heatMaps {
				vector := make([]float64, 0)
				for _, user := range users {
					ftr := user.Model.Grid(r, c)[ftrType]
					vector = append(vector, ftr.Value())
				}
				heatMaps[ftrType].Set(r, c, stat.Variance(vector, nil))
			}
		}
	}

	return heatMaps
}

func variationWithinUser(id int) map[features.FeatureType]*mat.Dense {
	heatMaps := map[features.FeatureType]*mat.Dense{
		features.LengthFeatureType:   mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.GradientFeatureType: mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.HOGFeatureType:      mat.NewDense(*flags.Rows, *flags.Cols, nil),
		features.CornersFeatureType:  mat.NewDense(*flags.Rows, *flags.Cols, nil),
	}

	userModel := cmd.EnrollUser(uint8(id), samplesUsers[id], uint16(*flags.Rows), uint16(*flags.Cols))
	for r := 0; r < *flags.Rows; r++ {
		for c := 0; c < *flags.Cols; c++ {
			for ftrType := range heatMaps {
				ftr := userModel.Model.Grid(r, c)[ftrType]
				heatMaps[ftrType].Set(r, c, ftr.Var())
			}
		}
	}

	return heatMaps
}

func main() {
	flag.IntVar(&userId, "u", -1, "user to generate variation within user's samples, "+
		"set to negative or leave on default for variation between users")
	flag.BoolVar(&fullResources, "full", false, "run test on full dataset")
	flag.BoolVar(&addTimestamp, "time", false, "add test time to filenames")
	flag.Parse()
	flags.AreaFilterOff = newTrue()
	flags.StdMeanFilterOff = newTrue()

	start = time.Now()
	if addTimestamp {
		startString = start.Format(TestStartTimeFormat) + "_"
	}

	samplesUsers = cmd.GenuineUsers(fullResources)
	if _, ok := samplesUsers[userId]; userId > 0 && !ok {
		log.Fatal("No such user")
	}

	if userId > 0 {
		start := time.Now()

		dir := path.Join("res", fmt.Sprintf("user%03d", userId))
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Fatal(err)
		}

		heatMaps := variationWithinUser(userId)

		elapsed := time.Since(start)
		if flags.Verbose() {
			log.Printf("Calculated variations within user %03d in %s\n", userId, elapsed)
		}

		for ft, hm := range heatMaps {
			if mat.Min(hm) >= mat.Max(hm) {
				log.Println(fmt.Sprintf("empty matrix %s of values %f", ft, mat.Min(hm)))
				continue
			}
			canvas, err := plotHeatmap(hm, fmt.Sprintf("%s Variation Within User %d", ft, userId))
			if err != nil {
				log.Fatal(err)
			}
			if err := savePlot(canvas, dir, startString+ft.String()+".png"); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		start := time.Now()

		heatMaps := variationBetweenUsers()

		elapsed := time.Since(start)
		if flags.Verbose() {
			log.Printf("Calculated variations between users in %s\n", elapsed)
		}

		for ft, hm := range heatMaps {
			if mat.Min(hm) >= mat.Max(hm) {
				log.Println(fmt.Sprintf("empty matrix %s of values %f", ft, mat.Min(hm)))
				continue
			}
			canvas, err := plotHeatmap(hm, fmt.Sprintf("%s Variation Between Users", ft))
			if err != nil {
				log.Fatal(err)
			}
			if err := savePlot(canvas, "res", startString+ft.String()+".png"); err != nil {
				log.Fatal(err)
			}
		}
	}
}
