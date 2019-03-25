package auth

import (
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/stat"
	"math"
)

type FeatureType string

const (
	Basic        FeatureType = "BasicFeatures"
	LengthGrid   FeatureType = "LengthGridFeatures"
	GradientGrid FeatureType = "GradientGridFeatures"
)

type Feature struct {
	Std    float64
	Mean   float64
	Values []float64
}

func NewFeature() *Feature {
	return &Feature{
		Std:    0.0,
		Mean:   0.0,
		Values: nil,
	}
}

type Features interface {
	Score(filename string) float64
	Extract(filename string)
}

type BasicFeatures map[string]*Feature

func NewBasicFeatures() *BasicFeatures {
	return &BasicFeatures{
		"aspect":   NewFeature(),
		"length":   NewFeature(),
		"gradient": NewFeature(),
	}
}

func (f BasicFeatures) Score(filename string) float64 {
	pattern := *NewBasicFeatures()
	sample := NewImage(filename)
	sample.Preprocess()
	pattern.Extract(*sample, 1)

	scores := make([]float64, 0)
	for key := range f {
		//fmt.Printf("%s: pattern: %f, mean: %f, std: %f\n", key, pattern[key].Mean, f[key].Mean, f[key].Std)
		score := stat.StdScore(pattern[key].Mean, f[key].Mean, f[key].Std)
		scores = append(scores, score)
	}
	return stat.Mean(scores, nil)
}

func (f BasicFeatures) Extract(sample Image, nSamples int) {
	sample.Update()
	aspect := sample.Ratio()

	length := float64(gocv.CountNonZero(sample.Mat()))

	sobelX := gocv.NewMat()
	sobelY := gocv.NewMat()
	gocv.SpatialGradient(sample.Mat(), &sobelX, &sobelY, 3, gocv.BorderReplicate)
	gradX := gocv.CountNonZero(sobelX)
	gradY := gocv.CountNonZero(sobelY)
	gradient := 0.0
	if gradX != 0 && gradY != 0 {
		gradient = float64(gradX) / float64(gradX+gradY)
	}

	if nSamples != 1 {
		weights := []float64{float64(nSamples - 1), 1}

		values := []float64{f["aspect"].Mean, aspect}
		f["aspect"].Mean, f["aspect"].Std = stat.MeanStdDev(values, weights)

		values = []float64{f["length"].Mean, length}
		f["length"].Mean, f["length"].Std = stat.MeanStdDev(values, weights)

		values = []float64{f["gradient"].Mean, gradient}
		f["gradient"].Mean, f["gradient"].Std = stat.MeanStdDev(values, weights)
	} else {
		f["aspect"].Mean = aspect
		f["length"].Mean = length
		f["gradient"].Mean = gradient
	}
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
