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
	rows  int
	cols  int
}

func NewFeatures(rows, cols int) *Features {
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
	return &Features{
		basic: FeatureMap{
			"aspect":   NewFeature("aspect", aspect),
			"length":   NewFeature("length", length),
			"gradient": NewFeature("gradient", gradient),
		},
		grid: g,
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

func (f *Features) Score(sample *samples.Sample) (float64, float64, *Features) {
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
	for i := range gss {
		gss[i] = make([]float64, f.cols)
		for j := range gss[i] {
			ss := make([]float64, 0)
			for key, template := range f.grid[i][j] {
				s := stat.StdScore(pattern.grid[i][j][key].mean, template.mean, template.std)
				ss = append(ss, math.Abs(s))
			}
			gss[i][j] = stat.Mean(ss, nil)
		}
	}
	rowScores := make([]float64, f.rows)
	for i, row := range gss {
		rowScores[i] = stat.Mean(row, nil)
	}
	gridScore := stat.Mean(rowScores, nil)

	return basicScore, gridScore, pattern
}

func (f *Features) Extract(sample *samples.Sample, nSamples int) {
	sample.Update()

	var wg sync.WaitGroup
	wg.Add(len(f.basic))

	for _, ftr := range f.basic {
		go func(ftr *Feature) {
			defer wg.Done()
			ftr.Update(sample, nSamples)
		}(ftr)
	}

	sampleGrid := samples.NewSampleGrid(sample, f.rows, f.cols)
	for i := range f.grid {
		for j := range f.grid[i] {
			wg.Add(len(f.grid[i][j]))
			for _, ftr := range f.grid[i][j] {
				go func(ftr *Feature) {
					defer wg.Done()
					ftr.Update(sampleGrid.At(i, j), nSamples)
				}(ftr)
			}
		}
	}

	wg.Wait()
}
