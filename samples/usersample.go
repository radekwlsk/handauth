package samples

import (
	"fmt"
	"gocv.io/x/gocv"
)

type UserSample struct {
	user   string
	sample *Sample
}

func (s *UserSample) Sample() *Sample {
	return s.sample
}

func NewSignature(username, filename string) UserSample {
	im := NewSample(filename)
	s := UserSample{
		user:   username,
		sample: im,
	}
	return s
}

func (s *UserSample) Preprocess() {
	s.sample.Preprocess(0.0)
}

func (s *UserSample) Show() {
	window := gocv.NewWindow(fmt.Sprintf("%s's signature (%dx%d)", s.user, s.sample.Width(), s.sample.Height()))
	defer window.Close()
	window.ResizeWindow(s.sample.Width(), s.sample.Height())
	window.IMShow(s.sample.Mat())
	for window.IsOpen() {
		if window.WaitKey(1) > 0 {
			break
		}
	}
}
