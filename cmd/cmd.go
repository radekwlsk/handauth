package cmd

import (
	"fmt"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/samples"
	"github.com/radekwlsk/handauth/signature"
	"io/ioutil"
	"path"
	"strconv"
)

const ResourcesFullGenuinePath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/genuines/full/"
const ResourcesTestGenuinePath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/genuines/test/"
const ResourcesFullForgeryPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/forgeries/full/"
const ResourcesTestForgeryPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/forgeries/test/"
const FileNameFormat = "NFI-%03d%02d%03d.png"

var UseFullResources = true

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
	sample, err := samples.NewUserSample(userName, filePath)
	if err != nil {
		return nil, err
	} else {
		return sample, nil
	}
}

func EnrollUser(id uint8, samplesIds []int, rows, cols uint16) signature.UserModel {
	template := signature.NewModel(rows, cols, nil)
	ok := false
	var sample *samples.UserSample
	var err error
	for i, s := range samplesIds {
		sample, err = ReadUserSample(id, id, uint8(s))
		if err != nil {
			continue
		} else {
			ok = true
		}
		sample.Preprocess()
		template.Extract(sample.Sample(), i+1)
		sample.Close()
	}
	if !ok {
		return signature.UserModel{
			Id: id,
		}
	}
	if !*flags.AreaFilterOff {
		_ = template.AreaFilter(*flags.AreaFilterFieldThreshold, *flags.AreaFilterRowColThreshold)
	}
	if !*flags.StdMeanFilterOff {
		_ = template.StdMeanFilter(*flags.StdMeanFilterThreshold)
	}
	return signature.UserModel{
		Id:    id,
		Model: template,
	}
}

func EnrollUserSync(id uint8, samplesIds []int, rows, cols uint16, users chan *signature.UserModel) {
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

func scoreSample(id, i uint8, template *signature.UserModel) (signature.Score, error) {
	sample, err := ReadUserSample(id, template.Id, i)
	if err != nil {
		return nil, err
	}
	sample.Preprocess()
	score, _ := template.Model.Score(sample.Sample())
	sample.Close()
	return score, nil
}

func VerifyUser(
	id uint8,
	samplesIds []int,
	template *signature.UserModel,
	thresholds []float64,
	thresholdWeights map[signature.AreaType]float64,
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
	template *signature.UserModel,
	thresholds []float64,
	thresholdWeights map[signature.AreaType]float64,
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

func GenuineUsers(full bool) map[int][]int {
	genuineSamplesUsers := make(map[int][]int)
	var genuinePath string
	if full {
		genuinePath = ResourcesFullGenuinePath
	} else {
		genuinePath = ResourcesTestGenuinePath
	}
	files, err := ioutil.ReadDir(genuinePath)
	if err != nil {
		panic("couldn't read files")
	}
	for _, f := range files {
		if !f.IsDir() {
			user, err := strconv.Atoi(f.Name()[4:7])
			if err != nil {
				panic("wrong user id position")
			}
			sample, err := strconv.Atoi(f.Name()[7:9])
			if err != nil {
				panic("wrong sample id position")
			}
			if _, ok := genuineSamplesUsers[user]; ok {
				genuineSamplesUsers[user] = append(genuineSamplesUsers[user], sample)
			} else {
				genuineSamplesUsers[user] = []int{sample}
			}
		}
	}
	return genuineSamplesUsers
}

func ForgeryUsers(full bool) map[[2]int][]int {
	forgerySamplesUsers := make(map[[2]int][]int)
	var forgeryPath string
	if full {
		forgeryPath = ResourcesFullForgeryPath
	} else {
		forgeryPath = ResourcesTestForgeryPath
	}
	files, err := ioutil.ReadDir(forgeryPath)
	if err != nil {
		panic("couldn't read files")
	}
	for _, f := range files {
		if !f.IsDir() {
			sample, err := strconv.Atoi(f.Name()[7:9])
			if err != nil {
				panic("wrong sample id position")
			}
			forger, err := strconv.Atoi(f.Name()[4:7])
			if err != nil {
				panic("wrong forger id position")
			}
			user, err := strconv.Atoi(f.Name()[9:12])
			if err != nil {
				panic("wrong user id position")
			}
			key := [2]int{forger, user}
			if _, ok := forgerySamplesUsers[key]; ok {
				forgerySamplesUsers[key] = append(forgerySamplesUsers[key], sample)
			} else {
				forgerySamplesUsers[key] = []int{sample}
			}
		}
	}
	return forgerySamplesUsers
}
