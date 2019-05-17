package features

import (
	"fmt"
	"github.com/radekwlsk/handauth/samples"
	"gonum.org/v1/gonum/stat"
	"math"
	"strings"
)

var FeatureFlags = map[FeatureType]bool{
	LengthFeatureType:   true,
	GradientFeatureType: true,
	AspectFeatureType:   true,
	HOGFeatureType:      true,
	CornersFeatureType:  true,
}

type FeatureType int

func (t FeatureType) String() string {
	return []string{
		"LengthFeature",
		"GradientFeature",
		"AspectFeature",
		"HOGFeature",
		"CornersFeature",
	}[t]
}

const (
	LengthFeatureType FeatureType = iota
	GradientFeatureType
	AspectFeatureType
	HOGFeatureType
	CornersFeatureType
)

type Feature struct {
	fType    FeatureType
	std      float64
	mean     float64
	variance float64
	max      float64
	min      float64
	function func(sample *samples.Sample) float64
}

func (f *Feature) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s %.3f", f.fType, f.mean))
	if f.min != f.max {
		sb.WriteString(fmt.Sprintf(" [%.3f, %.3f]", f.min, f.max))
	}
	if f.variance != 0 {
		sb.WriteString(fmt.Sprintf(", var %.3f(%.3f)", f.variance, f.std))
	}
	return sb.String()
}

func (f *Feature) Update(sample *samples.Sample, nSamples int) {
	value := f.function(sample)

	switch nSamples {
	case 0:
		panic("nSamples has to be at least 1 - for first sample enroll")
	case 1:
		f.mean = value
		f.min = value
		f.max = value
		f.variance = 0.0
		f.std = 0.0
		break
	default:
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

func (f *Feature) Value() float64 {
	return f.mean
}

func (f *Feature) Std() float64 {
	return f.std
}

func (f *Feature) Score(other *Feature) float64 {
	return stat.StdScore(other.Value(), f.mean, f.std)
}

type FeatureMap map[FeatureType]*Feature

func (m FeatureMap) GoString() string {
	var ftrStrings []string
	for ftrType, ftr := range m {
		if FeatureFlags[ftrType] {
			ftrStrings = append(ftrStrings, ftr.String())
		}
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}
