package features

import (
	"fmt"
	"github.com/radekwlsk/handauth/samples"
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/stat"
	"math"
	"strings"
)

type FeatureType string

const (
	Basic FeatureType = "Features"
	Grid  FeatureType = "GridFeatures"
)

type Feature struct {
	name     string
	std      float64
	mean     float64
	variance float64
	max      float64
	min      float64
	function func(sample *samples.Sample) float64
}

type FeatureMap = map[string]*Feature

func NewFeature(name string, function func(sample *samples.Sample) float64) *Feature {
	return &Feature{name: name, function: function}
}

func (f *Feature) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<Feature %s: %.3f", f.name, f.mean))
	if f.min != f.max {
		sb.WriteString(fmt.Sprintf(" [%.3f, %.3f]", f.min, f.max))
	}
	if f.variance != 0 {
		sb.WriteString(fmt.Sprintf(", var %.3f(%.3f)", f.variance, f.std))
	}
	sb.WriteString(">")
	return sb.String()
}

func (f *Feature) Name() string {
	return f.name
}

func (f *Feature) Update(sample *samples.Sample, nSamples int) {
	value := f.function(sample)
	if nSamples == 1 {
		f.mean = value
		f.min = value
		f.max = value
		f.variance = 0.0
		f.std = 0.0
	} else {
		newMean := stat.Mean([]float64{f.mean, value}, []float64{float64(nSamples - 1), 1})
		newVar := f.variance + (math.Pow(value-f.mean, 2) / float64(nSamples))
		newVar *= float64(nSamples-1) / float64(nSamples)

		f.mean = newMean
		f.variance = newVar
		f.std = math.Sqrt(newVar)

		if value > f.max {
			f.max = value
		}
		if value < f.min {
			f.min = value
		}
	}
}

func aspect(sample *samples.Sample) float64 {
	return sample.Ratio()
}

func length(sample *samples.Sample) float64 {
	return float64(gocv.CountNonZero(sample.Mat()))
}

func gradient(sample *samples.Sample) float64 {
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
}
