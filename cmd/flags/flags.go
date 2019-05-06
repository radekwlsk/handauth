package flags

import (
	"flag"
	"fmt"
	"strconv"
)

const (
	ColsDefault                = 10
	RowsDefault                = 4
	MinThresholdDefault        = 1.0
	MaxThresholdDefault        = 3.0
	ThresholdStepDefault       = 0.1
	BasicThresholdScaleDefault = 1.0
	GridThresholdScaleDefault  = 1.0
	RowThresholdScaleDefault   = 1.0
	ColThresholdScaleDefault   = 1.0
)

var (
	verbose             = flag.Bool("v", false, "print basic messages")
	VVerbose            = flag.Bool("vv", false, "print additional execution messages")
	Cols                = flag.Int("cols", ColsDefault, "columns in grid")
	Rows                = flag.Int("rows", RowsDefault, "rows in grid")
	minThreshold        = flag.Float64("min", MinThresholdDefault, "test threshold min value")
	maxThreshold        = flag.Float64("max", MaxThresholdDefault, "test threshold max value")
	thresholdStep       = flag.Float64("step", ThresholdStepDefault, "test threshold step value")
	basicThresholdScale = flag.Float64("basic-scale", BasicThresholdScaleDefault,
		"test threshold scale for basic score")
	gridThresholdScale = flag.Float64("grid-scale", GridThresholdScaleDefault,
		"test threshold scale for grid score")
	rowThresholdScale = flag.Float64("row-scale", RowThresholdScaleDefault,
		"test threshold scale for row score")
	colThresholdScale = flag.Float64("col-scale", ColThresholdScaleDefault,
		"test threshold scale for col score")
)

func Thresholds() []float64 {
	var thresholds []float64
	var threshold float64
	for i := 0; threshold < *maxThreshold; i++ {
		threshold = *minThreshold + (*thresholdStep * float64(i))
		threshold, _ = strconv.ParseFloat(fmt.Sprintf("%.3f", threshold), 64)
		thresholds = append(thresholds, threshold)
	}
	return thresholds
}

func ThresholdWeights() []float64 {
	return []float64{
		*basicThresholdScale,
		*gridThresholdScale,
		*rowThresholdScale,
		*colThresholdScale,
	}
}

func Verbose() bool {
	return *verbose || *VVerbose
}
