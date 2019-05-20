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

var AreaFlags = map[AreaType]bool{
	BasicAreaType: true,
	RowAreaType:   true,
	ColAreaType:   true,
	GridAreaType:  true,
}

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

func (model *Model) Basic() features.FeatureMap {
	return model.basic
}

func (model *Model) Grid(r, c int) features.FeatureMap {
	return model.grid[[2]int{r, c}]
}

func (model *Model) Row(r int) features.FeatureMap {
	return model.row[r]
}

func (model *Model) Col(c int) features.FeatureMap {
	return model.col[c]
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
			//features.HOGFeatureType:      features.NewHOGFeature(),
			features.AspectFeatureType:      features.NewAspectFeature(),
			features.MassCenterXFeatureType: features.NewMassCenterFeature(features.XMassCenter),
			features.MassCenterYFeatureType: features.NewMassCenterFeature(features.YMassCenter),
		}
	}
	if AreaFlags[GridAreaType] {
		grid = make(GridFeatureMap)
		for _, rc := range gridKeys {
			grid[rc] = features.FeatureMap{
				features.LengthFeatureType:   features.NewLengthFeature(),
				features.HOGFeatureType:      features.NewHOGFeature(),
				features.GradientFeatureType: features.NewGradientFeature(),
				features.CornersFeatureType:  features.NewCornersFeature(),
			}
		}
	}
	if AreaFlags[RowAreaType] {
		row = make(RowFeatureMap)
		for _, r := range rowKeys {
			row[r] = features.FeatureMap{
				features.LengthFeatureType: features.NewLengthFeature(),
				//features.HOGFeatureType:      features.NewHOGFeature(),
				features.GradientFeatureType: features.NewGradientFeature(),
				features.CornersFeatureType:  features.NewCornersFeature(),
			}
		}
	}
	if AreaFlags[ColAreaType] {
		col = make(ColFeatureMap)
		for _, c := range colKeys {
			col[c] = features.FeatureMap{
				features.LengthFeatureType: features.NewLengthFeature(),
				//features.HOGFeatureType:      features.NewHOGFeature(),
				features.GradientFeatureType: features.NewGradientFeature(),
				features.CornersFeatureType:  features.NewCornersFeature(),
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

func (model *Model) GoString() string {
	var sb strings.Builder
	if AreaFlags[BasicAreaType] {
		sb.WriteString(fmt.Sprintf("\t%#v\n", model.basic))
	}
	if AreaFlags[GridAreaType] {
		sb.WriteString(fmt.Sprintf("\t%#v\n", model.grid))
	}
	if AreaFlags[RowAreaType] {
		sb.WriteString(fmt.Sprintf("\t%#v\n", model.row))
	}
	if AreaFlags[ColAreaType] {
		sb.WriteString(fmt.Sprintf("\t%#v\n", model.col))
	}
	return fmt.Sprintf("<%T \n%s>", model, sb.String())
}

func (model *Model) FieldsCount() int {
	return len(model.grid)
}

func (model *Model) RowsCount() int {
	return len(model.row)
}

func (model *Model) ColsCount() int {
	return len(model.col)
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

func (model *Model) getScoreFunc(area AreaType) (func(ftr1, ftr2 *Model) float64, bool) {
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

func (model *Model) Score(sample *samples.Sample) (Score, *Model) {
	pattern := NewModel(model.rows, model.cols, model)
	pattern.Extract(sample, 1)

	score := make(Score)

	for area, flag := range AreaFlags {
		if scoreFunc, ok := model.getScoreFunc(area); flag && ok {
			score[area] = scoreFunc(model, pattern)
		}
	}

	return score, pattern
}

func (model *Model) Extract(sample *samples.Sample, nSamples int) {
	sample.Update()

	for ftrType, ftr := range model.basic {
		if features.FeatureFlags[ftrType] {
			ftr.Update(sample, nSamples)
		}
	}

	sampleGrid := samples.NewSampleGrid(sample, model.rows, model.cols)
	model.gridConfig = sampleGrid.Config()
	for rc, ftrMap := range model.grid {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				s := sampleGrid.At(rc[0], rc[1])
				ftr.Update(s, nSamples)
				s.Close()
			}
		}
	}

	for r, ftrMap := range model.row {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				s := sampleGrid.At(r, -1)
				ftr.Update(s, nSamples)
				s.Close()
			}
		}
	}

	for c, ftrMap := range model.col {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] {
				s := sampleGrid.At(-1, c)
				ftr.Update(s, nSamples)
				s.Close()
			}
		}
	}
}

func (model *Model) AreaFilter(fieldThreshold float64, rowColThreshold float64) error {
	if model.gridConfig == (samples.GridConfig{}) {
		return fmt.Errorf("at least one sample has to be extracted before filtering")
	}
	if !features.FeatureFlags[features.LengthFeatureType] {
		return nil
	}

	fieldAreaLimit := model.gridConfig.FieldArea() * fieldThreshold
	rowAreaLimit := model.gridConfig.RowArea() * rowColThreshold
	colAreaLimit := model.gridConfig.ColArea() * rowColThreshold

	for rc, ftrMap := range model.grid {
		lnFtr := ftrMap[features.LengthFeatureType]
		if lnFtr.Value() < fieldAreaLimit {
			delete(model.grid, rc)
		}
	}

	for r, ftrMap := range model.row {
		lnFtr := ftrMap[features.LengthFeatureType]
		if lnFtr.Value() < rowAreaLimit {
			delete(model.row, r)
		}
	}

	for c, ftrMap := range model.col {
		lnFtr := ftrMap[features.LengthFeatureType]
		if lnFtr.Value() < colAreaLimit {
			delete(model.col, c)
		}
	}
	return nil
}

func (model *Model) StdMeanFilter(threshold float64) error {
	if model.gridConfig == (samples.GridConfig{}) {
		return fmt.Errorf("at least one sample has to be extracted before filtering")
	}

	for rc, ftrMap := range model.grid {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] && ftr.Std() > ftr.Value()*threshold {
				delete(model.grid, rc)
				break
			}
		}
	}

	for r, ftrMap := range model.row {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] && ftr.Std() > ftr.Value()*threshold {
				delete(model.row, r)
				break
			}
		}
	}

	for c, ftrMap := range model.col {
		for ftrType, ftr := range ftrMap {
			if features.FeatureFlags[ftrType] && ftr.Std() > ftr.Value()*threshold {
				delete(model.col, c)
				break
			}
		}
	}
	return nil
}
