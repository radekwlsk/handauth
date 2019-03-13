package features

import (
	"github.com/radekwlsk/handauth/auth"
	"github.com/radekwlsk/handauth/utils"
)

type FeatureType string

const (
	Basic          FeatureType = "BasicFeatures"
	LengthGradient FeatureType = "LengthGradientFeatures"
)

type Features interface {
	Score(template auth.Template, sample utils.Image) float64
}

type BasicFeatures struct {
	aspect      float64
	length      float64
	avgLength   float64
	gradient    float64
	avgGradient float64
}

func (f *BasicFeatures) Score(template auth.Template, sample utils.Image) float64 {
	panic("not implemented")
}

type LengthGradientFeatures struct {
	gridSize [2]int
}

func (f *LengthGradientFeatures) Score(template auth.Template, sample utils.Image) float64 {
	panic("not implemented")
}
