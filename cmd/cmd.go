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

const ResourcesSigCompFullGenuinePath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/genuines/full/"
const ResourcesSigCompTestGenuinePath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/genuines/test/"
const ResourcesSigCompFullForgeryPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/forgeries/full/"
const ResourcesSigCompTestForgeryPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/forgeries/test/"
const ResourcesGPDSPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/gpds/"
const FileNameFormatSigComp = "NFI-%03d%02d%03d.png"
const FileNameFormatGPDS = "%s-%03d-%02d.jpg"

type ResourceType string

const (
	GPDSResources    ResourceType = "GPDS"
	SigCompResources ResourceType = "SigComp"
)

var UseFullResources = true
var Resources = SigCompResources

func readSigCompUserSample(creator, user uint16, index uint8) (*samples.UserSample, error) {
	var resPath string
	if creator == user {
		if UseFullResources {
			resPath = ResourcesSigCompFullGenuinePath
		} else {
			resPath = ResourcesSigCompTestGenuinePath
		}
	} else {
		if UseFullResources {
			resPath = ResourcesSigCompFullForgeryPath
		} else {
			resPath = ResourcesSigCompTestForgeryPath
		}
	}
	filePath := path.Join(resPath, fmt.Sprintf(FileNameFormatSigComp, creator, index, user))
	userName := fmt.Sprintf("%02d", user)
	sample, err := samples.NewUserSample(userName, filePath)
	if err != nil {
		return nil, err
	} else {
		return sample, nil
	}
}

func readGPDSUserSample(user uint16, index uint8, forgery bool) (*samples.UserSample, error) {
	var prefix string
	if forgery {
		prefix = "cf"
	} else {
		prefix = "c"
	}
	filePath := path.Join(
		ResourcesGPDSPath,
		fmt.Sprintf("%03d", user),
		fmt.Sprintf(FileNameFormatGPDS, prefix, user, index),
	)
	userName := fmt.Sprintf("%04d", user)
	sample, err := samples.NewUserSample(userName, filePath)
	if err != nil {
		return nil, err
	} else {
		return sample, nil
	}
}

func ReadUserSample(creator, user uint16, index uint8) (*samples.UserSample, error) {
	switch Resources {
	case SigCompResources:
		return readSigCompUserSample(creator, user, index)
	case GPDSResources:
		forgery := creator != user
		return readGPDSUserSample(user, index, forgery)
	default:
		panic(fmt.Sprintf("no such resources: %v", Resources))
	}
}

func EnrollUser(id uint16, samplesIds []int, rows, cols uint16) signature.UserModel {
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

func EnrollUserSync(id uint16, samplesIds []int, rows, cols uint16, users chan *signature.UserModel) {
	uf := EnrollUser(id, samplesIds, rows, cols)
	users <- &uf
	return
}

type VerificationResult struct {
	TemplateUserId uint16
	SampleUserId   uint16
	SuccessCounts  []uint8
	RejectedCounts []uint8
}

func scoreSample(id uint16, i uint8, template *signature.UserModel) (signature.Score, error) {
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
	id uint16,
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
	id uint16,
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
	switch Resources {
	case SigCompResources:
		var genuinePath string
		if full {
			genuinePath = ResourcesSigCompFullGenuinePath
		} else {
			genuinePath = ResourcesSigCompTestGenuinePath
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
		break
	case GPDSResources:
		for i := 0; i < *flags.GPDSUsers; i++ {
			ss := make([]int, 24)
			for i := range ss {
				ss[i] = i + 1
			}
			genuineSamplesUsers[i+1] = ss
		}
	}
	return genuineSamplesUsers
}

func ForgeryUsers(full bool) map[[2]int][]int {
	forgerySamplesUsers := make(map[[2]int][]int)
	switch Resources {
	case SigCompResources:
		var forgeryPath string
		if full {
			forgeryPath = ResourcesSigCompFullForgeryPath
		} else {
			forgeryPath = ResourcesSigCompTestForgeryPath
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
		break
	case GPDSResources:
		ss := make([]int, 30)
		for i := range ss {
			ss[i] = i + 1
		}
		for i := 0; i < *flags.GPDSUsers; i++ {
			forgerySamplesUsers[[2]int{i, i + 1}] = ss
		}
	}
	return forgerySamplesUsers
}
