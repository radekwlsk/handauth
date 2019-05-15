package features

import (
	"fmt"
	"github.com/radekwlsk/handauth/samples"
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/stat"
	"math"
)

func NewHOGFeature() *Feature {
	return &Feature{fType: HOGFeatureType, function: histogramOfGradients}
}

func histogramOfGradients(sample *samples.Sample) float64 {
	if sample.Empty() {
		panic(fmt.Sprintf("empty mat in %#v", sample))
	}
	sobelX := gocv.NewMat()
	sobelY := gocv.NewMat()
	defer sobelX.Close()
	defer sobelY.Close()
	gocv.Sobel(sample.Mat(), &sobelX, gocv.MatTypeCV32F, 1, 0, 3, 1.0, 0.0, gocv.BorderReplicate)
	gocv.Sobel(sample.Mat(), &sobelY, gocv.MatTypeCV32F, 0, 1, 3, 1.0, 0.0, gocv.BorderReplicate)
	magnitude := gocv.NewMat()
	angle := gocv.NewMat()
	defer magnitude.Close()
	defer angle.Close()
	gocv.CartToPolar(sobelX, sobelY, &magnitude, &angle, true)
	bins := make(map[int]float64)
	for r := 0; r < angle.Rows(); r++ {
		for c := 0; c < angle.Cols(); c++ {
			a := math.Mod(float64(angle.GetFloatAt(r, c)), 180.0)
			if a < 0 {
				a = a + 180.0
			}
			b := int(math.RoundToEven(a))
			bins[b] = bins[b] + float64(magnitude.GetFloatAt(r, c))
		}
	}
	var total float64
	for _, m := range bins {
		total += m
	}
	weights := make([]float64, 90)
	values := make([]float64, 90)
	for i := 0; i < 90; i++ {
		if m, ok := bins[i*2]; ok {
			weights[i] = m / total
		}
		values[i] = float64(i)
	}
	return stat.Mean(values, weights)
}
