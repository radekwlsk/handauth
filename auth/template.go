package auth

import (
	"github.com/radekwlsk/handauth/features"
	"time"
)

type Template struct {
	user              string
	updateTime        time.Time
	nSamples          int
	width             int
	gridStride        float64
	gridSize          [2]float64
	features          *features.Features
	featuresEnabled   map[features.AreaType]bool
	featuresAvailable map[features.AreaType]bool
}

func NewTemplate(
	user string,
	width int,
	gridStride float64,
	gridSize [2]float64,
) *Template {
	t := Template{
		user:       user,
		updateTime: time.Now(),
		nSamples:   0,
		width:      width,
		gridStride: gridStride,
		gridSize:   gridSize,
	}
	return &t
}
