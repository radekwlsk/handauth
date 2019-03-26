package auth

import (
	"fmt"
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/stat"
	"math"
	"sync"
)

type FeatureType string

const (
	Basic        FeatureType = "BasicFeatures"
	LengthGrid   FeatureType = "LengthGridFeatures"
	GradientGrid FeatureType = "GradientGridFeatures"
)

type Feature struct {
	Name     string
	Std      float64
	Mean     float64
	Var      float64
	Max      float64
	Min      float64
	Function func(sample *Image) float64
}

func (f *Feature) String() string {
	return fmt.Sprintf("%s: [%.3f, %.3f], ~%.3f, var %.3f(%.3f)", f.Name, f.Min, f.Max, f.Mean, f.Var, f.Std)
}

func (f *Feature) Update(sample *Image, nSamples int) {
	value := f.Function(sample)
	if nSamples == 1 {
		f.Mean = value
		f.Min = value
		f.Max = value
		f.Var = 0.0
		f.Std = 0.0
	} else {
		newMean := stat.Mean([]float64{f.Mean, value}, []float64{float64(nSamples - 1), 1})
		newVar := f.Var + (math.Pow(value-f.Mean, 2) / float64(nSamples))
		newVar *= float64(nSamples-1) / float64(nSamples)

		f.Mean = newMean
		f.Var = newVar
		f.Std = math.Sqrt(newVar)

		if value > f.Max {
			f.Max = value
		}
		if value < f.Min {
			f.Min = value
		}
	}
}

type Features interface {
	Score(filename string) float64
	Extract(filename string)
}

type BasicFeatures map[string]*Feature

func NewBasicFeatures() *BasicFeatures {
	return &BasicFeatures{
		"aspect": &Feature{
			Name: "aspect",
			Function: func(sample *Image) float64 {
				return sample.Ratio()
			},
		},
		"length": &Feature{
			Name: "length",
			Function: func(sample *Image) float64 {
				return float64(gocv.CountNonZero(sample.Mat()))
			},
		},
		"gradient": &Feature{
			Name: "gradient",
			Function: func(sample *Image) float64 {
				sobelX := gocv.NewMat()
				sobelY := gocv.NewMat()
				gocv.SpatialGradient(sample.Mat(), &sobelX, &sobelY, 3, gocv.BorderReplicate)
				gradX := gocv.CountNonZero(sobelX)
				gradY := gocv.CountNonZero(sobelY)
				if gradX != 0 && gradY != 0 {
					return float64(gradX) / float64(gradX+gradY)
				} else {
					return 0.0
				}
			},
		},
	}
}

func (f BasicFeatures) Score(filename string) (float64, *BasicFeatures) {
	pattern := NewBasicFeatures()
	sample := NewImage(filename)
	sample.Preprocess()
	pattern.Extract(sample, 1)

	scores := make([]float64, 0)
	for key := range f {
		//fmt.Printf("%s: pattern: %f, mean: %f, std: %f\n", key, pattern[key].Mean, f[key].Mean, f[key].Std)
		score := stat.StdScore((*pattern)[key].Mean, f[key].Mean, f[key].Std)
		scores = append(scores, math.Abs(score))
	}
	return stat.Mean(scores, nil), pattern
}

func (f BasicFeatures) Extract(sample *Image, nSamples int) {
	sample.Update()

	var wg sync.WaitGroup
	wg.Add(len(f))

	for _, feature := range f {
		go func(f *Feature) {
			defer wg.Done()
			f.Update(sample, nSamples)
		}(feature)
	}

	wg.Wait()
}

type LengthGridFeatures struct {
	gridSize  [2]int
	stride    int
	avgLength [3]float64
	lengths   [][][3]float64
}

func NewLengthGridFeatures(x, y, stride int) *LengthGridFeatures {
	f := &LengthGridFeatures{
		gridSize:  [2]int{x, y},
		stride:    stride,
		avgLength: [3]float64{0.0, 0.0, math.MaxFloat64},
		lengths:   [][][3]float64{},
	}
	f.lengths = make([][][3]float64, x)
	for i := range f.lengths {
		f.lengths[i] = make([][3]float64, y)
		for j := range f.lengths[i] {
			f.lengths[i][j] = [3]float64{0.0, 0.0, math.MaxFloat64}
		}
	}
	return f
}

func (f *LengthGridFeatures) Score(template Template, sample Image) float64 {
	panic("not implemented")
}

func (f *LengthGridFeatures) Extract(sample Image) {

}
