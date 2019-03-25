package auth

import (
	"time"
)

type Template struct {
	user              string
	updateTime        time.Time
	width             int
	nSamples          int
	features          map[FeatureType]Features
	featuresEnabled   map[FeatureType]bool
	featuresAvailable map[FeatureType]bool
}

func NewTemplate(user string, width int) *Template {
	t := Template{
		user:       user,
		updateTime: time.Now(),
		width:      width,
		nSamples:   0,
		features:   map[FeatureType]Features{},
	}
	return &t
}
