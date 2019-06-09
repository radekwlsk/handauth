package flags

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/signature"
	"strconv"
)

const (
	ColsDefault                      = 60
	RowsDefault                      = 20
	MinThresholdDefault              = 0.5
	MaxThresholdDefault              = 5.5
	ThresholdStepDefault             = 0.02
	BasicThresholdScaleDefault       = 1.0
	GridThresholdScaleDefault        = 1.0
	RowThresholdScaleDefault         = 1.0
	ColThresholdScaleDefault         = 1.0
	AreaFilterFieldThresholdDefault  = 0.03
	AreaFilterRowColThresholdDefault = 0.02
	StdFilterThresholdDefault        = 0.5
)

var (
	GPDSUsers           = flag.Int("gpds", 100, "amount of GPDS users to use if flag res = 1")
	Resources           = flag.Int("res", 0, "resources type 0 - SigComp, 1 - GPDS, 2 - MCYT")
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
	AreaFilterOff            = flag.Bool("no-area-filter", false, "turn area filter off")
	AreaFilterFieldThreshold = flag.Float64("area-filter-field", AreaFilterFieldThresholdDefault,
		"area filter field min threshold")
	AreaFilterRowColThreshold = flag.Float64("area-filter-rowcol", AreaFilterRowColThresholdDefault,
		"area filter row/col min threshold")
	StdFilterOff       = flag.Bool("no-std-filter", false, "turn std-mean filter off")
	StdFilterThreshold = flag.Float64("std-filter", StdFilterThresholdDefault,
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

func ThresholdWeights() map[signature.AreaType]float64 {
	return map[signature.AreaType]float64{
		signature.BasicAreaType: *basicThresholdScale,
		signature.GridAreaType:  *gridThresholdScale,
		signature.RowAreaType:   *rowThresholdScale,
		signature.ColAreaType:   *colThresholdScale,
	}
}

func Verbose() bool {
	return *verbose || *VVerbose
}
