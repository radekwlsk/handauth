package auth

import (
	"fmt"
	"gocv.io/x/gocv"
)

type Signature struct {
	user  string
	image *Image
}

func (s *Signature) Image() *Image {
	return s.image
}

func NewSignature(username, filename string) Signature {
	im := NewImage(filename)
	s := Signature{
		user:  username,
		image: im,
	}
	return s
}

func (s *Signature) Preprocess() {
	s.image.Preprocess()
}

func (s *Signature) Show() {
	window := gocv.NewWindow(fmt.Sprintf("%s's signature (%dx%d)", s.user, s.image.Width(), s.image.Height()))
	defer window.Close()
	window.ResizeWindow(s.image.Width(), s.image.Height())
	window.IMShow(s.image.Mat())
	for window.IsOpen() {
		if window.WaitKey(1) > 0 {
			break
		}
	}
}
