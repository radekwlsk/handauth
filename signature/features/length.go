package features

import (
	"github.com/radekwlsk/handauth/samples"
	"gocv.io/x/gocv"
)

func NewLengthFeature() *Feature {
	return &Feature{fType: LengthFeatureType, function: length}
}

func length(sample *samples.Sample) float64 {
	return float64(gocv.CountNonZero(sample.Mat()))
}
