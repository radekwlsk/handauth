package features

import (
	"fmt"
	"github.com/radekwlsk/handauth/samples"
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/stat"
	"math"
	"strings"
)

type AreaType int

func (t AreaType) String() string {
	return []string{
		"BasicArea",
		"RowArea",
		"ColArea",
		"GridArea",
	}[t]
}

const (
	BasicAreaType AreaType = iota
	RowAreaType
	ColAreaType
	GridAreaType
)

type FeatureType int

func (t FeatureType) String() string {
	return []string{
		"LengthFeature",
		"GradientFeature",
		"AspectFeature",
	}[t]
}

const (
	LengthFeatureType FeatureType = iota
	GradientFeatureType
	AspectFeatureType
)

type FeatureMap map[FeatureType]*Feature
type GridFeatureMap map[[2]int]FeatureMap
type RowFeatureMap map[int]FeatureMap
type ColFeatureMap map[int]FeatureMap

func (m FeatureMap) GoString() string {
	var ftrStrings []string
	for _, ftr := range m {
		ftrStrings = append(ftrStrings, ftr.String())
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}

func (m GridFeatureMap) GoString() string {
	var ftrStrings []string
	for rc, ftrMap := range m {
		ftrStrings = append(ftrStrings, fmt.Sprintf("[%d,%d] %#v", rc[0], rc[1], ftrMap))
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}

func (m RowFeatureMap) GoString() string {
	var ftrStrings []string
	for r, ftrMap := range m {
		ftrStrings = append(ftrStrings, fmt.Sprintf("[%d] %#v", r, ftrMap))
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}

func (m ColFeatureMap) GoString() string {
	var ftrStrings []string
	for c, ftrMap := range m {
		ftrStrings = append(ftrStrings, fmt.Sprintf("[%d] %#v", c, ftrMap))
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}

func NewLengthFeature() *Feature {
	return &Feature{fType: LengthFeatureType, function: length}
}

func NewGradientFeature() *Feature {
	return &Feature{fType: GradientFeatureType, function: gradient}
}

func NewAspectFeature() *Feature {
	return &Feature{fType: AspectFeatureType, function: aspect}
}

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

func aspect(sample *samples.Sample) float64 {
	return sample.Ratio()
}

func length(sample *samples.Sample) float64 {
	return float64(gocv.CountNonZero(sample.Mat()))
}

func gradient(sample *samples.Sample) float64 {
	if sample.Empty() {
		panic(fmt.Sprintf("empty mat in %#v", sample))
	}
	sobelX := gocv.NewMat()
	sobelY := gocv.NewMat()
	defer sobelX.Close()
	defer sobelY.Close()
	gocv.SpatialGradient(sample.Mat(), &sobelX, &sobelY, 3, gocv.BorderReplicate)
	gradX := gocv.CountNonZero(sobelX)
	gradY := gocv.CountNonZero(sobelY)
	if gradX != 0 && gradY != 0 {
		return float64(gradX) / float64(gradX+gradY)
	} else {
		return 0.0
	}
}

//
//func histogramOfGradients(sample *samples.Sample) float64 {
//	if sample.Empty() {
//		panic(fmt.Sprintf("empty mat in %#v", sample))
//	}
//	sobelX := gocv.NewMat()
//	sobelY := gocv.NewMat()
//	defer sobelX.Close()
//	defer sobelY.Close()
//	gocv.SpatialGradient(sample.Mat(), &sobelX, &sobelY, 3, gocv.BorderReplicate)
//	magnitude := gocv.NewMat()
//	angle := gocv.NewMat()
//	defer magnitude.Close()
//	defer angle.Close()
//	gocv.CartToPolar(sobelX, sobelY, &magnitude, &angle, true)
//	bins := map[int]float64{0: .0, 20: .0, 40: .0, 60: .0, 80: .0, 100: .0, 120: .0, 140: .0, 160: .0}
//	for r := 0; r < angle.Rows(); r++ {
//		for c := 0; c < angle.Cols(); c++ {
//
//		}
//	}
//
//
//	return float64(gocv.CountNonZero(sample.Mat()))
//}
