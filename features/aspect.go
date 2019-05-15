package features

import "github.com/radekwlsk/handauth/samples"

func NewAspectFeature() *Feature {
	return &Feature{fType: AspectFeatureType, function: aspect}
}

func aspect(sample *samples.Sample) float64 {
	return sample.Ratio()
}
