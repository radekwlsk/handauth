package auth

import (
	"github.com/radekwlsk/handauth/features"
	"time"
)

type Template struct {
	user              string
	updateTime        time.Time
	width             int
	nSamples          int
	features          map[features.FeatureType]features.Features
	featuresEnabled   map[features.FeatureType]bool
	featuresAvailable map[features.FeatureType]bool
}

func NewTemplate(user string, width int) *Template {
	t := Template{
		user:       user,
		updateTime: time.Now(),
		width:      width,
		nSamples:   0,
		features:   map[features.FeatureType]features.Features{},
	}
	return &t
}
