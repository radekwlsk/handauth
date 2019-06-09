package cmd

import (
	"fmt"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/samples"
	"github.com/radekwlsk/handauth/signature"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const ResourcesSigCompPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/sigcomp/"
const FileNameFormatSigComp = "NFI-%03d%02d%03d.png"

const ResourcesGPDSPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/gpds/"
const FileNameFormatGPDS = "%s-%03d-%02d.jpg"

const ResourcesMCYTPath = "/home/radoslaw/go/src/github.com/radekwlsk/handauth/res/mcyt/"
const FileNameFormatMCYT = "%04d%s%02d.bmp"

type ResourceType int

const (
	SigCompResources ResourceType = iota
	GPDSResources
	MCYTResources
)

var UseFullResources = true

func readSigCompUserSample(creator, user uint16, index uint8) (*samples.UserSample, error) {
	resPath := ResourcesSigCompPath
	if creator == user {
		resPath = path.Join(resPath, "genuines")
	} else {
		resPath = path.Join(resPath, "forgeries")
	}
	if UseFullResources {
		resPath = path.Join(resPath, "full")
	} else {
		resPath = path.Join(resPath, "test")
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

func readMCYTUserSample(creator, user uint16, index uint8) (*samples.UserSample, error) {
	resPath := path.Join(ResourcesMCYTPath, fmt.Sprintf("%04d", user))
	var prefix, label string
	if creator == user {
		label = "v"
		prefix = ""
	} else {
		label = "f"
		prefix = fmt.Sprintf("%04d_", creator)
	}
	filePath := path.Join(resPath, prefix+fmt.Sprintf(FileNameFormatMCYT, user, label, index))
	userName := fmt.Sprintf("%04d", user)
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
	switch ResourceType(*flags.Resources) {
	case SigCompResources:
		return readSigCompUserSample(creator, user, index)
	case GPDSResources:
		forgery := creator != user
		return readGPDSUserSample(user, index, forgery)
	case MCYTResources:
		return readMCYTUserSample(creator, user, index)
	default:
		panic(fmt.Sprintf("no such resources: %v", ResourceType(*flags.Resources)))
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
	if !*flags.StdFilterOff {
		_ = template.StdFilter(*flags.StdFilterThreshold)
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
	switch ResourceType(*flags.Resources) {
	case SigCompResources:
		genuinePath := path.Join(ResourcesSigCompPath, "genuines")
		if full {
			genuinePath = path.Join(genuinePath, "full")
		} else {
			genuinePath = path.Join(genuinePath, "test")
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
	case MCYTResources:
		err := filepath.Walk(ResourcesMCYTPath,
			func(path string, f os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !f.IsDir() {
					if strings.Contains(f.Name(), "v") {
						user, err := strconv.Atoi(f.Name()[0:4])
						if err != nil {
							panic("wrong user id position")
						}
						sample, err := strconv.Atoi(f.Name()[5:7])
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
				return nil
			})
		if err != nil {
			panic("couldn't read files")
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
	switch ResourceType(*flags.Resources) {
	case SigCompResources:
		forgeryPath := path.Join(ResourcesSigCompPath, "forgeries")
		if full {
			forgeryPath = path.Join(forgeryPath, "full")
		} else {
			forgeryPath = path.Join(forgeryPath, "test")
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
	case MCYTResources:
		err := filepath.Walk(ResourcesMCYTPath,
			func(path string, f os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !f.IsDir() {
					if strings.Contains(f.Name(), "f") {
						forger, err := strconv.Atoi(f.Name()[0:4])
						if err != nil {
							panic("wrong forger id position")
						}
						user, err := strconv.Atoi(f.Name()[5:9])
						if err != nil {
							panic("wrong user id position")
						}
						sample, err := strconv.Atoi(f.Name()[10:12])
						if err != nil {
							panic("wrong sample id position")
						}
						key := [2]int{forger, user}
						if _, ok := forgerySamplesUsers[key]; ok {
							forgerySamplesUsers[key] = append(forgerySamplesUsers[key], sample)
						} else {
							forgerySamplesUsers[key] = []int{sample}
						}
					}
				}
				return nil
			})
		if err != nil {
			panic("couldn't read files")
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
