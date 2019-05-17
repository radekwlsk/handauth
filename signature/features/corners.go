package features

import (
	"fmt"
	"github.com/radekwlsk/handauth/samples"
	"gocv.io/x/gocv"
)

func NewCornersFeature() *Feature {
	return &Feature{fType: CornersFeatureType, function: corners}
}

func corners(sample *samples.Sample) float64 {
	if sample.Empty() {
		panic(fmt.Sprintf("empty mat in %#v", sample))
	}
	corners := gocv.NewMat()
	defer corners.Close()

	gocv.GoodFeaturesToTrack(sample.Mat(), &corners, 0, 0.01, 5)

	lines := float64(corners.Rows())
	return lines
}
