package features

import (
	"fmt"
	"strings"
)

var AreaFlags = map[AreaType]bool{
	BasicAreaType: true,
	RowAreaType:   true,
	ColAreaType:   true,
	GridAreaType:  true,
}

type AreaType int

func (t AreaType) String() string {
	return []string{
		"BasicArea",
		"RowArea",
		"ColArea",
		"GridArea",
	}[t]
}

const (
	BasicAreaType AreaType = iota
	RowAreaType
	ColAreaType
	GridAreaType
)

type FeatureMap map[FeatureType]*Feature
type GridFeatureMap map[[2]int]FeatureMap
type RowFeatureMap map[int]FeatureMap
type ColFeatureMap map[int]FeatureMap

func (m FeatureMap) GoString() string {
	var ftrStrings []string
	for _, ftr := range m {
		ftrStrings = append(ftrStrings, ftr.String())
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}

func (m GridFeatureMap) GoString() string {
	var ftrStrings []string
	for rc, ftrMap := range m {
		ftrStrings = append(ftrStrings, fmt.Sprintf("[%d,%d] %#v", rc[0], rc[1], ftrMap))
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}

func (m RowFeatureMap) GoString() string {
	var ftrStrings []string
	for r, ftrMap := range m {
		ftrStrings = append(ftrStrings, fmt.Sprintf("[%d] %#v", r, ftrMap))
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}

func (m ColFeatureMap) GoString() string {
	var ftrStrings []string
	for c, ftrMap := range m {
		ftrStrings = append(ftrStrings, fmt.Sprintf("[%d] %#v", c, ftrMap))
	}
	return fmt.Sprintf("<%T %s>", m, strings.Join(ftrStrings, ", "))
}
