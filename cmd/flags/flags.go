package flags

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/features"
	"strconv"
)

const (
	ColsDefault                      = 60
	RowsDefault                      = 20
	MinThresholdDefault              = 1.0
	MaxThresholdDefault              = 3.0
	ThresholdStepDefault             = 0.05
	BasicThresholdScaleDefault       = 1.0
	GridThresholdScaleDefault        = 1.0
	RowThresholdScaleDefault         = 1.0
	ColThresholdScaleDefault         = 1.0
	AreaFilterFieldThresholdDefault  = 0.05
	AreaFilterRowColThresholdDefault = 0.02
	StdMeanFilterThresholdDefault    = 0.5
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
	AreaFilterFieldThreshold = flag.Float64("filter-area-field", AreaFilterFieldThresholdDefault,
		"area filter field min threshold")
	AreaFilterRowColThreshold = flag.Float64("filter-area-rowcol", AreaFilterRowColThresholdDefault,
		"area filter row/col min threshold")
	StdMeanFilterThreshold = flag.Float64("filter-std", StdMeanFilterThresholdDefault,
		"std-mean filter max threshold")
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

func ThresholdWeights() map[features.AreaType]float64 {
	return map[features.AreaType]float64{
		features.BasicAreaType: *basicThresholdScale,
		features.GridAreaType:  *gridThresholdScale,
		features.RowAreaType:   *rowThresholdScale,
		features.ColAreaType:   *colThresholdScale,
	}
}

func Verbose() bool {
	return *verbose || *VVerbose
}
