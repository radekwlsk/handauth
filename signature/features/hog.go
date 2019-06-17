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
	} else if gocv.CountNonZero(sample.Mat()) == 0 {
		return 0.0
	}
	sobelX := gocv.NewMat()
	sobelY := gocv.NewMat()
	gocv.Sobel(sample.Mat(), &sobelX, gocv.MatTypeCV32F,
		1, 0, 3, 1.0, 0.0, gocv.BorderReplicate)
	gocv.Sobel(sample.Mat(), &sobelY, gocv.MatTypeCV32F,
		0, 1, 3, 1.0, 0.0, gocv.BorderReplicate)
	mgMat := gocv.NewMat()
	agMat := gocv.NewMat()
	gocv.CartToPolar(sobelX, sobelY, &mgMat, &agMat, true)
	sobelX.Close()
	sobelY.Close()
	magnitude, err := mgMat.DataPtrFloat32()
	if err != nil {
		panic(err)
	}
	angle, err := agMat.DataPtrFloat32()
	if err != nil {
		panic(err)
	}
	rows := agMat.Rows()
	cols := agMat.Cols()
	mgMat.Close()
	agMat.Close()
	bins := make(map[int]float64)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			m := magnitude[r*(cols)+c]
			if m != 0 {
				a := math.Mod(float64(angle[r*(cols)+c]), 180.0)
				if a < 0 {
					a = a + 180.0
				}
				b := int(math.Floor(a / 2.0))
				bins[b] = bins[b] + float64(m)
			}
		}
	}
	var total float64
	for _, m := range bins {
		total += m
	}
	weights := make([]float64, 90)
	values := make([]float64, 90)
	for i := 0; i < 90; i++ {
		if total > 0 {
			weights[i] = bins[i] / total
		} else {
			weights[i] = 0.0
		}
		values[i] = float64(i * 2)
	}
	value := stat.Mean(values, weights)
	return value
}
