package signature

import (
	"fmt"
	"github.com/radekwlsk/handauth/signature/features"
	"strings"
)

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

type GridFeatureMap map[[2]int]features.FeatureMap
type RowFeatureMap map[int]features.FeatureMap
type ColFeatureMap map[int]features.FeatureMap

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
