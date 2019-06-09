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

func plotHeatmap(data *mat.Dense, title string) (*plot.Plot, error) {
	m := Grid{
		Data: data,
	}
	pal, err := brewer.GetPalette(brewer.TypeSequential, "BuPu", 9)
	if err != nil {
		return nil, err
	}
	h := plotter.NewHeatMap(m, pal)

	p, err := plot.New()
	if err != nil {
		return nil, err
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
	l.Top = true
	l.YOffs = -p.Title.Font.Extents().Height
	p.Legend = l

	return p, nil
}

func savePlot(p *plot.Plot, dir, filename string) error {
	err := p.Save(720, 480, path.Join(dir, filename))
	if err != nil {
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

	var models []*signature.UserModel
	{
		featuresChan := make(chan *signature.UserModel)

		for user, samples := range samplesUsers {
			for _, sampleId := range samples {
				go cmd.EnrollUserSync(uint16(user), []int{sampleId}, uint16(*flags.Rows), uint16(*flags.Cols),
					featuresChan)
			}
		}

		for _, samples := range samplesUsers {
			for range samples {
				f := <-featuresChan
				if f.Model != nil {
					models = append(models, f)
				}
			}
		}
		close(featuresChan)
	}

	for r := 0; r < *flags.Rows; r++ {
		for c := 0; c < *flags.Cols; c++ {
			for ftrType := range heatMaps {
				vector := make([]float64, len(models))
				for i, model := range models {
					ftr := model.Model.Grid(r, c)[ftrType]
					vector[i] = ftr.Value()
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

	userModel := cmd.EnrollUser(uint16(id), samplesUsers[id], uint16(*flags.Rows), uint16(*flags.Cols))
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
	flags.StdFilterOff = newTrue()

	start = time.Now()
	if addTimestamp {
		startString = start.Format(TestStartTimeFormat) + "_"
	}

	if userId > 0 {
		fullResources = true
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
			plt, err := plotHeatmap(hm, fmt.Sprintf("%s Variation Within User %03d", ft, userId))
			if err != nil {
				log.Fatal(err)
			}
			filename := startString + fmt.Sprintf("user%03d", userId) + ft.String() + ".png"
			if err = plt.Save(720, 480, path.Join(dir, filename)); err != nil {
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
			plt, err := plotHeatmap(hm, fmt.Sprintf("%s Variation Between Users", ft))
			if err != nil {
				log.Fatal(err)
			}
			filename := startString + ft.String() + ".png"
			if err = plt.Save(720, 480, path.Join("res", filename)); err != nil {
				log.Fatal(err)
			}
		}
	}
}
