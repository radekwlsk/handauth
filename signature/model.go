package signature

import (
	"fmt"
	"github.com/radekwlsk/handauth/samples"
	"github.com/radekwlsk/handauth/signature/features"
	"gonum.org/v1/gonum/stat"
	"log"
	"math"
	"os"
	"strings"
)

var Debug = false
var logger = log.New(os.Stdout, "[features] ", log.Lshortfile+log.Ltime)

type UserModel struct {
	Id    uint8
	Model *Model
}

type Model struct {
	basic      features.FeatureMap
	grid       GridFeatureMap
	row        RowFeatureMap
	col        ColFeatureMap
	rows       uint16
	cols       uint16
	gridConfig samples.GridConfig
}

func NewModel(rows, cols uint16, template *Model) *Model {
	var rowKeys, colKeys []int
	var gridKeys [][2]int
	if template == nil {
		rowKeys = make([]int, rows)
		for i := 0; i < int(rows); i++ {
			rowKeys[i] = i
		}
		colKeys = make([]int, cols)
		for i := 0; i < int(cols); i++ {
			colKeys[i] = i
		}
		for _, r := range rowKeys {
			for _, c := range colKeys {
				gridKeys = append(gridKeys, [2]int{r, c})
			}
		}
	} else {
		for r := range template.row {
			rowKeys = append(rowKeys, r)
		}
		for c := range template.col {
			colKeys = append(colKeys, c)
		}
		for rc := range template.grid {
			gridKeys = append(gridKeys, rc)
		}
	}
	return newModel(rows, cols, rowKeys, colKeys, gridKeys)
}

func newModel(rows, cols uint16, rowKeys, colKeys []int, gridKeys [][2]int) *Model {
	var basic features.FeatureMap
	var grid GridFeatureMap
	var row RowFeatureMap
	var col ColFeatureMap

	if AreaFlags[BasicAreaType] {
		basic = features.FeatureMap{
			features.LengthFeatureType:   features.NewLengthFeature(),
			features.GradientFeatureType: features.NewGradientFeature(),
			features.HOGFeatureType:      features.NewHOGFeature(),
			features.AspectFeatureType:   features.NewAspectFeature(),
		}
	}
	if AreaFlags[GridAreaType] {
		grid = make(GridFeatureMap)
		for _, rc := range gridKeys {
			grid[rc] = features.FeatureMap{
				features.LengthFeatureType:   features.NewLengthFeature(),
				features.HOGFeatureType:      features.NewHOGFeature(),
				features.GradientFeatureType: features.NewGradientFeature(),
			}
		}
	}
	if AreaFlags[RowAreaType] {
		row = make(RowFeatureMap)
		for _, r := range rowKeys {
			row[r] = features.FeatureMap{
				features.LengthFeatureType:   features.NewLengthFeature(),
				features.HOGFeatureType:      features.NewHOGFeature(),
				features.GradientFeatureType: features.NewGradientFeature(),
			}
		}
	}
	if AreaFlags[ColAreaType] {
		col = make(ColFeatureMap)
		for _, c := range colKeys {
			col[c] = features.FeatureMap{
				features.LengthFeatureType:   features.NewLengthFeature(),
				features.HOGFeatureType:      features.NewHOGFeature(),
				features.GradientFeatureType: features.NewGradientFeature(),
			}
		}
	}
	return &Model{
		basic: basic,
		grid:  grid,
		row:   row,
		col:   col,
		rows:  rows,
		cols:  cols,
	}
}

func (f *Model) GoString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\t%#v\n", f.basic))
	sb.WriteString(fmt.Sprintf("\t%#v\n", f.grid))
	sb.WriteString(fmt.Sprintf("\t%#v\n", f.row))
	sb.WriteString(fmt.Sprintf("\t%#v\n", f.col))
	return fmt.Sprintf("<%T \n%s>", f, sb.String())
}

func (f *Model) FieldsCount() int {
	return len(f.grid)
}

func (f *Model) RowsCount() int {
	return len(f.row)
}

func (f *Model) ColsCount() int {
	return len(f.col)
}

type Score map[AreaType]float64

type AreaThresholdWeights map[AreaType]float64

func (s Score) Check(t float64, weights AreaThresholdWeights) (bool, error) {
	var weight float64
	for area, score := range s {
		if w, ok := weights[area]; ok {
			weight = w
		} else {
			weight = 1.0
		}
		if (score * weight) >= t {
			return false, nil
		}
	}
	return true, nil
}

func scoreBasic(t, s *Model) float64 {
	ss := make([]float64, 0)
	for ftrType, ftr := range t.basic {
		if features.FeatureFlags[ftrType] {
			if Debug {
				logger.Printf("score basic %s: sample: %s, template: %s\n",
					ftrType, s.basic[ftrType], ftr)
			}
			s := ftr.Score(s.basic[ftrType])
			ss = append(ss, math.Abs(s))
		}
	}
	return stat.Mean(ss, nil)
}

func scoreGrid(t, s *Model) float64 {
	gss := make([]float64, len(t.grid))
	for rc, ftrMap := range t.grid {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				if Debug {
					logger.Printf("score grid (%d,%d) %s: sample: %s, template: %s\n",
						rc[0], rc[1], ftrType, s.grid[rc][ftrType], ftr)
				}
				s := ftr.Score(s.grid[rc][ftrType])
				gss = append(gss, math.Abs(s))
			}
		}
	}
	return stat.Mean(gss, nil)
}

func scoreRow(t, s *Model) float64 {
	rss := make([]float64, len(t.row))
	for r, ftrMap := range t.row {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				if Debug {
					logger.Printf("score row %d %s: sample: %s, template: %s\n",
						r, ftrType, s.row[r][ftrType], ftr)
				}
				s := ftr.Score(s.row[r][ftrType])
				rss = append(rss, math.Abs(s))
			}
		}
	}
	return stat.Mean(rss, nil)
}

func scoreCol(t, s *Model) float64 {
	css := make([]float64, len(t.col))
	for c, ftrMap := range t.col {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				if Debug {
					logger.Printf("score col %d %s: sample: %s, template: %s\n",
						c, ftrType, s.col[c][ftrType], ftr)
				}
				s := ftr.Score(s.col[c][ftrType])
				css = append(css, math.Abs(s))
			}
		}
	}
	return stat.Mean(css, nil)
}

func (f *Model) getScoreFunc(area AreaType) (func(ftr1, ftr2 *Model) float64, bool) {
	switch area {
	case BasicAreaType:
		return scoreBasic, true
	case GridAreaType:
		return scoreGrid, true
	case RowAreaType:
		return scoreRow, true
	case ColAreaType:
		return scoreCol, true
	default:
		return nil, false
	}
}

func (f *Model) Score(sample *samples.Sample) (Score, *Model) {
	pattern := NewModel(f.rows, f.cols, f)
	pattern.Extract(sample, 1)

	score := make(Score)

	for area, flag := range AreaFlags {
		if scoreFunc, ok := f.getScoreFunc(area); flag && ok {
			score[area] = scoreFunc(f, pattern)
		}
	}

	return score, pattern
}

func (f *Model) Extract(sample *samples.Sample, nSamples int) {
	sample.Update()

	for ftrType, ftr := range f.basic {
		if features.FeatureFlags[ftrType] {
			ftr.Update(sample, nSamples)
		}
	}

	sampleGrid := samples.NewSampleGrid(sample, f.rows, f.cols)
	f.gridConfig = sampleGrid.Config()
	for rc, ftrMap := range f.grid {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				ftr.Update(sampleGrid.At(rc[0], rc[1]), nSamples)
			}
		}
	}

	for r, ftrMap := range f.row {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				ftr.Update(sampleGrid.At(r, -1), nSamples)
			}
		}
	}

	for c, ftrMap := range f.col {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				ftr.Update(sampleGrid.At(-1, c), nSamples)
			}
		}
	}
}

func (f *Model) AreaFilter(fieldThreshold float64, rowColThreshold float64) error {
	if f.gridConfig == (samples.GridConfig{}) {
		return fmt.Errorf("at least one sample has to be extracted before filtering")
	}

	fieldAreaLimit := f.gridConfig.FieldArea() * fieldThreshold
	rowAreaLimit := f.gridConfig.RowArea() * rowColThreshold
	colAreaLimit := f.gridConfig.ColArea() * rowColThreshold

	for rc, ftrMap := range f.grid {
		lnFtr := ftrMap[features.LengthFeatureType]
		if lnFtr.Value() < fieldAreaLimit {
			delete(f.grid, rc)
		}
	}

	for r, ftrMap := range f.row {
		lnFtr := ftrMap[features.LengthFeatureType]
		if lnFtr.Value() < rowAreaLimit {
			delete(f.row, r)
		}
	}

	for c, ftrMap := range f.col {
		lnFtr := ftrMap[features.LengthFeatureType]
		if lnFtr.Value() < colAreaLimit {
			delete(f.col, c)
		}
	}
	return nil
}

func (f *Model) StdMeanFilter(threshold float64) error {
	if f.gridConfig == (samples.GridConfig{}) {
		return fmt.Errorf("at least one sample has to be extracted before filtering")
	}

	for rc, ftrMap := range f.grid {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] && ftr.Std() > ftr.Value()*threshold {
				delete(f.grid, rc)
				break
			}
		}
	}

	for r, ftrMap := range f.row {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] && ftr.Std() > ftr.Value()*threshold {
				delete(f.row, r)
				break
			}
		}
	}

	for c, ftrMap := range f.col {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] && ftr.Std() > ftr.Value()*threshold {
				delete(f.col, c)
				break
			}
		}
	}
	return nil
}

var AreaFlags = map[AreaType]bool{
	BasicAreaType: true,
	RowAreaType:   true,
	ColAreaType:   true,
	GridAreaType:  true,
}
