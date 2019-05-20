package features

import (
	"github.com/radekwlsk/handauth/samples"
)

const XMassCenter = 0
const YMassCenter = 1

func NewMassCenterFeature(pos int) *Feature {
	return &Feature{fType: MassCenterXFeatureType, function: massCenter(pos)}
}

func massCenter(pos int) func(sample *samples.Sample) float64 {
	if pos == XMassCenter {
		return func(sample *samples.Sample) float64 {
			cm := sample.CenterOfMass()
			return float64(cm.X) / float64(sample.Width())

		}
	} else if pos == YMassCenter {
		return func(sample *samples.Sample) float64 {
			cm := sample.CenterOfMass()
			return float64(cm.Y) / float64(sample.Height())
		}
	} else {
		panic("")
	}
}
