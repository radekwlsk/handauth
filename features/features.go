package features

import (
	"fmt"
	"github.com/radekwlsk/handauth/samples"
	"gonum.org/v1/gonum/stat"
	"math"
	"strings"
	"sync"
)

type Features struct {
	basic FeatureMap
	grid  [][]FeatureMap
	row   []FeatureMap
	col   []FeatureMap
	rows  uint8
	cols  uint8
}

func NewFeatures(rows, cols uint8) *Features {
	g := make([][]FeatureMap, rows)
	for i := range g {
		g[i] = make([]FeatureMap, cols)
		for j := range g[i] {
			g[i][j] = FeatureMap{
				"length":   NewFeature("length", length),
				"gradient": NewFeature("gradient", gradient),
			}
		}
	}
	r := make([]FeatureMap, rows)
	for i := range r {
		r[i] = FeatureMap{
			"length": NewFeature("length", length),
		}
	}
	c := make([]FeatureMap, cols)
	for i := range c {
		c[i] = FeatureMap{
			"length": NewFeature("length", length),
		}
	}
	return &Features{
		basic: FeatureMap{
			"aspect":   NewFeature("aspect", aspect),
			"length":   NewFeature("length", length),
			"gradient": NewFeature("gradient", gradient),
		},
		grid: g,
		row:  r,
		col:  c,
		rows: rows,
		cols: cols,
	}
}

func (f *Features) String() string {
	var sb strings.Builder
	for _, ftr := range f.basic {
		sb.WriteString(ftr.String())
	}
	return fmt.Sprintf("<Features %s>", sb.String())
}

type Score struct {
	basic float64
	grid  float64
	row   float64
	col   float64
}

func (s *Score) Basic() float64 {
	return s.basic
}

func (s *Score) Grid() float64 {
	return s.grid
}

func (s *Score) Row() float64 {
	return s.row
}

func (s *Score) Col() float64 {
	return s.col
}

func (s *Score) Check(t float64, weights []float64) (bool, error) {
	if weights == nil {
		weights = []float64{1.0, 1.0, 1.0, 1.0}
	} else if len(weights) != 4 {
		return false, fmt.Errorf("weights have to be nil or length 4: [basic, grid, row, col]")
	}
	for i, score := range []float64{s.basic, s.grid, s.row, s.col} {
		if (score * weights[i]) >= t {
			return false, nil
		}
	}
	return true, nil
}

func (f *Features) Score(sample *samples.Sample) (*Score, *Features) {
	pattern := NewFeatures(f.rows, f.cols)
	//sample.Resize(sample.Width(), 0.0)
	pattern.Extract(sample, 1)

	ss := make([]float64, 0)
	for key, template := range f.basic {
		//fmt.Printf("%s: pattern: %f, mean: %f, std: %f\n", key, pattern[key].mean, f[key].mean, f[key].std)
		s := stat.StdScore(pattern.basic[key].mean, template.mean, template.std)
		ss = append(ss, math.Abs(s))
	}
	basicScore := stat.Mean(ss, nil)

	gss := make([][]float64, f.rows)
	for r := range gss {
		gss[r] = make([]float64, f.cols)
		for c := range gss[r] {
			ss := make([]float64, 0)
			for key, template := range f.grid[r][c] {
				s := stat.StdScore(pattern.grid[r][c][key].mean, template.mean, template.std)
				ss = append(ss, math.Abs(s))
			}
			gss[r][c] = stat.Mean(ss, nil)
		}
	}
	gRowScores := make([]float64, f.rows)
	for r, row := range gss {
		gRowScores[r] = stat.Mean(row, nil)
	}
	gridScore := stat.Mean(gRowScores, nil)

	rss := make([]float64, f.rows)
	for r := range rss {
		ss := make([]float64, 0)
		for key, template := range f.row[r] {
			s := stat.StdScore(pattern.row[r][key].mean, template.mean, template.std)
			ss = append(ss, math.Abs(s))
		}
		rss[r] = stat.Mean(ss, nil)
	}
	rowScore := stat.Mean(rss, nil)

	css := make([]float64, f.cols)
	for c := range css {
		ss := make([]float64, 0)
		for key, template := range f.col[c] {
			s := stat.StdScore(pattern.col[c][key].mean, template.mean, template.std)
			ss = append(ss, math.Abs(s))
		}
		css[c] = stat.Mean(ss, nil)
	}
	colScore := stat.Mean(css, nil)

	return &Score{basicScore, gridScore, rowScore, colScore}, pattern
}

func (f *Features) Extract(sample *samples.Sample, nSamples int) {
	sample.Update()

	var wg sync.WaitGroup

	for _, ftr := range f.basic {
		wg.Add(1)
		go func(ftr *Feature) {
			defer wg.Done()
			ftr.Update(sample, nSamples)
		}(ftr)
	}

	sampleGrid := samples.NewSampleGrid(sample, f.rows, f.cols)
	for r := range f.grid {
		for c := range f.grid[r] {
			for _, ftr := range f.grid[r][c] {
				wg.Add(1)
				go func(ftr *Feature) {
					defer wg.Done()
					ftr.Update(sampleGrid.At(r, c), nSamples)
				}(ftr)
			}
		}
	}

	for r := range f.row {
		for _, ftr := range f.row[r] {
			wg.Add(1)
			go func(ftr *Feature) {
				defer wg.Done()
				ftr.Update(sampleGrid.At(r, -1), nSamples)
			}(ftr)
		}
	}

	for c := range f.col {
		for _, ftr := range f.col[c] {
			wg.Add(1)
			go func(ftr *Feature) {
				defer wg.Done()
				ftr.Update(sampleGrid.At(-1, c), nSamples)
			}(ftr)
		}
	}

	wg.Wait()
}
