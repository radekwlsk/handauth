package cmd

import (
	"fmt"
	"github.com/radekwlsk/handauth/features"
	"github.com/radekwlsk/handauth/samples"
	"path"
)

const ResourcesGenuinePath = "/home/radoslaw/pwr/mgr/thesis/sig_cv/res/NISDCC-offline/genuines/"
const ResourcesForgeryPath = "/home/radoslaw/pwr/mgr/thesis/sig_cv/res/NISDCC-offline/forgeries/"
const FileNameFormat = "NFI-%03d%02d%03d.png"

type UserFeatures struct {
	Id       uint8
	Features *features.Features
}

func ReadUserSample(creator, user, index uint8) (*samples.UserSample, error) {
	var resPath string
	if creator == user {
		resPath = ResourcesGenuinePath
	} else {
		resPath = ResourcesForgeryPath
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

func EnrollUser(id uint8, samplesIds []int, rows, cols uint8) UserFeatures {
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
	}
	//template.Filter(0.5)
	return UserFeatures{
		id,
		template,
	}
}

func EnrollUserSync(id uint8, samplesIds []int, rows, cols uint8, users chan *UserFeatures) {
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
