package cmd

import (
	"fmt"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/features"
	"github.com/radekwlsk/handauth/samples"
	"path"
)

const ResourcesFullGenuinePath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/genuines/full/"
const ResourcesTestGenuinePath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/genuines/test/"
const ResourcesFullForgeryPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/forgeries/full/"
const ResourcesTestForgeryPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/forgeries/test/"
const FileNameFormat = "NFI-%03d%02d%03d.png"

var UseFullResources = true

type UserFeatures struct {
	Id       uint8
	Features *features.Features
}

func ReadUserSample(creator, user, index uint8) (*samples.UserSample, error) {
	var resPath string
	if creator == user {
		if UseFullResources {
			resPath = ResourcesFullGenuinePath
		} else {
			resPath = ResourcesTestGenuinePath
		}
	} else {
		if UseFullResources {
			resPath = ResourcesFullForgeryPath
		} else {
			resPath = ResourcesTestForgeryPath
		}
	}
	filePath := path.Join(resPath, fmt.Sprintf(FileNameFormat, creator, index, user))
	userName := fmt.Sprintf("%02d", user)
	signature, err := samples.NewUserSample(userName, filePath)
	if err != nil {
		return nil, err
	} else {
		return signature, nil
	}
}

func EnrollUser(id uint8, samplesIds []int, rows, cols uint16) UserFeatures {
	template := features.NewFeatures(rows, cols, nil)
	ok := false
	var signature *samples.UserSample
	var err error
	for i, s := range samplesIds {
		signature, err = ReadUserSample(id, id, uint8(s))
		if err != nil {
			continue
		} else {
			ok = true
		}
		signature.Preprocess()
		template.Extract(signature.Sample(), i+1)
		signature.Close()
	}
	if !ok {
		template = nil
	} else {
		_ = template.AreaFilter(*flags.AreaFilterFieldThreshold, *flags.AreaFilterRowColThreshold)
		_ = template.StdMeanFilter(*flags.StdMeanFilterThreshold)
	}
	return UserFeatures{
		id,
		template,
	}
}

func EnrollUserSync(id uint8, samplesIds []int, rows, cols uint16, users chan *UserFeatures) {
	uf := EnrollUser(id, samplesIds, rows, cols)
	users <- &uf
	return
}

type VerificationResult struct {
	TemplateUserId uint8
	SampleUserId   uint8
	SuccessCounts  []uint8
	RejectedCounts []uint8
}

func scoreSample(id, i uint8, template *UserFeatures) (features.Score, error) {
	signature, err := ReadUserSample(id, template.Id, i)
	if err != nil {
		return nil, err
	}
	signature.Preprocess()
	score, _ := template.Features.Score(signature.Sample())
	signature.Close()
	return score, nil
}

func VerifyUser(
	id uint8,
	samplesIds []int,
	template *UserFeatures,
	thresholds []float64,
	thresholdWeights map[features.AreaType]float64,
) VerificationResult {
	successes := make([]uint8, len(thresholds))
	rejections := make([]uint8, len(thresholds))
	for _, s := range samplesIds {
		score, err := scoreSample(id, uint8(s), template)
		if err == nil {
			for i, t := range thresholds {
				success, err := score.Check(t, thresholdWeights)
				if err != nil {
					panic(err)
				}
				if success {
					successes[i] += 1
				} else {
					rejections[i] += 1
				}
			}
		}
	}
	return VerificationResult{
		template.Id,
		id,
		successes,
		rejections,
	}
}

func VerifyUserSync(
	id uint8,
	samplesIds []int,
	template *UserFeatures,
	thresholds []float64,
	thresholdWeights map[features.AreaType]float64,
	results chan *VerificationResult,
) {
	r := VerifyUser(id, samplesIds, template, thresholds, thresholdWeights)
	results <- &r
	return
}

type VerificationStat struct {
	PositiveCounts map[float64]uint16
	NegativeCounts map[float64]uint16
}

func (s *VerificationStat) AcceptanceRate(t float64) float64 {
	return float64(s.PositiveCounts[t]) / float64(s.Count(t))
}

func (s *VerificationStat) RejectionRate(t float64) float64 {
	return float64(s.NegativeCounts[t]) / float64(s.Count(t))
}

func (s *VerificationStat) Count(t float64) int {
	return int(s.PositiveCounts[t] + s.NegativeCounts[t])
}
