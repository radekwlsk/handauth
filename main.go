package main

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/features"
	"github.com/radekwlsk/handauth/samples"
	"path"
	"strconv"
)

const ResourcesGenuinePath = "/home/radoslaw/pwr/mgr/thesis/sig_cv/res/NISDCC-offline/genuines/"
const ResourcesForgeryPath = "/home/radoslaw/pwr/mgr/thesis/sig_cv/res/NISDCC-offline/forgeries/"
const FileNameFormat = "NFI-%03d%02d%03d.png"

const (
	Split               = 7
	Cols                = 10
	Rows                = 4
	MinThreshold        = 1.0
	MaxThreshold        = 3.0
	ThresholdStep       = 0.1
	BasicThresholdScale = 1.0
	GridThresholdScale  = 1.0
	RowThresholdScale   = 1.0
	ColThresholdScale   = 1.0
)

var verbose bool
var vVerbose bool
var cols int
var rows int
var split int
var minThreshold float64
var maxThreshold float64
var thresholdStep float64
var basicThresholdScale float64
var gridThresholdScale float64
var rowThresholdScale float64
var colThresholdScale float64

var thresholdWeights []float64

type UserFeatures struct {
	Id       uint8
	Features *features.Features
}

func readUserSample(creator, user, index uint8) (*samples.UserSample, error) {
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

func enrollUser(id uint8, users chan *UserFeatures) {
	template := features.NewFeatures(uint8(rows), uint8(cols))
	nSamples := 0
	ok := false
	var signature *samples.UserSample
	var err error
	for i := 0; i < split; i++ {
		signature, err = readUserSample(id, id, uint8(i))
		if err != nil {
			continue
		} else {
			ok = true
		}
		signature.Preprocess()
		nSamples++
		template.Extract(signature.Sample(), nSamples)
		signature.Close()
	}
	if !ok {
		template = nil
	}
	users <- &UserFeatures{
		id,
		template,
	}
	return
}

type VerificationResult struct {
	TemplateUserId uint8
	SampleUserId   uint8
	SuccessCounts  []uint8
	RejectedCounts []uint8
}

func scoreSample(id, i uint8, template *UserFeatures) (*features.Score, error) {
	signature, err := readUserSample(id, template.Id, i)
	if err != nil {
		return nil, err
	}
	signature.Preprocess()
	score, _ := template.Features.Score(signature.Sample())
	signature.Close()
	return score, nil
}

func verifyUser(id uint8, template *UserFeatures, thresholds []float64, results chan VerificationResult) {
	successes := make([]uint8, len(thresholds))
	rejections := make([]uint8, len(thresholds))
	start := 0
	if id == template.Id {
		start = split
	}
	for i := start; i <= 12; i++ {
		score, err := scoreSample(id, uint8(i), template)
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
	results <- VerificationResult{
		template.Id,
		id,
		successes,
		rejections,
	}
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

func main() {
	flag.BoolVar(&verbose, "v", false, "print basic messages")
	flag.BoolVar(&vVerbose, "vv", false, "print additional execution messages")
	flag.IntVar(&split, "split", Split, "enroll/test data split point")
	flag.IntVar(&cols, "cols", Cols, "columns in grid")
	flag.IntVar(&rows, "rows", Rows, "rows in grid")
	flag.Float64Var(&minThreshold, "min", MinThreshold, "test threshold min value")
	flag.Float64Var(&maxThreshold, "max", MaxThreshold, "test threshold max value")
	flag.Float64Var(&thresholdStep, "step", ThresholdStep, "test threshold step value")
	flag.Float64Var(&basicThresholdScale, "basic-scale", BasicThresholdScale, "test threshold scale for basic score")
	flag.Float64Var(&gridThresholdScale, "grid-scale", GridThresholdScale, "test threshold scale for grid score")
	flag.Float64Var(&rowThresholdScale, "row-scale", RowThresholdScale, "test threshold scale for row score")
	flag.Float64Var(&colThresholdScale, "col-scale", ColThresholdScale, "test threshold scale for col score")
	flag.Parse()

	verbose = verbose || vVerbose
	thresholdWeights = []float64{
		basicThresholdScale,
		gridThresholdScale,
		rowThresholdScale,
		colThresholdScale,
	}

	fmt.Printf(
		"%d\tcols\n"+
			"%d\trows\n"+
			"%d\textract/verify split\n"+
			"%v\tthreshold-weights\n",
		cols,
		rows,
		split,
		thresholdWeights,
	)

	thresholds := make([]float64, 0)
	var threshold float64
	for i := 0; threshold < maxThreshold; i++ {
		threshold = minThreshold + (thresholdStep * float64(i))
		threshold, _ = strconv.ParseFloat(fmt.Sprintf("%.3f", threshold), 64)
		thresholds = append(thresholds, threshold)
	}
	if verbose {
		fmt.Printf("%v\tthresholds\n", thresholds)
	}

	if verbose {
		fmt.Println("Started")
	}

	users := map[uint8]*UserFeatures{}
	featuresChan := make(chan *UserFeatures)

	for i := 1; i <= 100; i++ {
		go enrollUser(uint8(i), featuresChan)
	}

	for i := 1; i <= 100; i++ {
		f := <-featuresChan
		if f.Features != nil {
			users[f.Id] = f
			if vVerbose {
				fmt.Printf("\tEnrolled user %03d\n", f.Id)
			}
		}
	}

	close(featuresChan)
	if verbose {
		fmt.Printf("Enrolled %d users\n", len(users))
	}

	genuineResultsChan := make(chan VerificationResult)
	genuineStats := VerificationStat{
		PositiveCounts: map[float64]uint16{},
		NegativeCounts: map[float64]uint16{},
	}

	for id, user := range users {
		go verifyUser(id, user, thresholds, genuineResultsChan)
	}

	for range users {
		r := <-genuineResultsChan
		for i, t := range thresholds {
			if r.SuccessCounts[i]+r.RejectedCounts[i] > 0 {
				if _, ok := genuineStats.PositiveCounts[t]; ok {
					genuineStats.PositiveCounts[t] += uint16(r.SuccessCounts[i])
				} else {
					genuineStats.PositiveCounts[t] = uint16(r.SuccessCounts[i])
				}
				if _, ok := genuineStats.NegativeCounts[t]; ok {
					genuineStats.NegativeCounts[t] += uint16(r.RejectedCounts[i])
				} else {
					genuineStats.NegativeCounts[t] = uint16(r.RejectedCounts[i])
				}
			}
		}
		if vVerbose {
			fmt.Printf("\tVerified user %03d\n", r.TemplateUserId)
			for i, t := range thresholds {
				fmt.Printf(
					"\t\t%.2f: %d/%d\n",
					t,
					r.SuccessCounts[i],
					r.RejectedCounts[i],
				)
			}
		}
	}
	close(genuineResultsChan)
	if verbose {
		fmt.Printf("Verified all genuine users\n")
	}

	forgeriesResultsChan := make(chan VerificationResult)
	forgeriesStats := VerificationStat{
		PositiveCounts: map[float64]uint16{},
		NegativeCounts: map[float64]uint16{},
	}

	count := 0
	for _, user := range users {
		for i := 1; i <= 100; i++ {
			if uint8(i) != user.Id {
				go verifyUser(uint8(i), user, thresholds, forgeriesResultsChan)
				count += 1
			}
		}
	}

	for i := 0; i < count; i++ {
		r := <-forgeriesResultsChan
		for i, t := range thresholds {
			if r.SuccessCounts[i]+r.RejectedCounts[i] > 0 {
				if _, ok := forgeriesStats.PositiveCounts[t]; ok {
					forgeriesStats.PositiveCounts[t] += uint16(r.SuccessCounts[i])
				} else {
					forgeriesStats.PositiveCounts[t] = uint16(r.SuccessCounts[i])
				}
				if _, ok := forgeriesStats.NegativeCounts[t]; ok {
					forgeriesStats.NegativeCounts[t] += uint16(r.RejectedCounts[i])
				} else {
					forgeriesStats.NegativeCounts[t] = uint16(r.RejectedCounts[i])
				}
			}
		}
		if vVerbose {
			fmt.Printf(
				"\tVerified user %03d as %03d\n",
				r.SampleUserId,
				r.TemplateUserId,
			)
			for i, t := range thresholds {
				fmt.Printf(
					"\t\t%.2f: %d/%d\n",
					t,
					r.SuccessCounts[i],
					r.RejectedCounts[i],
				)
			}
		}
	}
	close(forgeriesResultsChan)
	if verbose {
		fmt.Printf("Verified all forgeries\n")
	}

	fmt.Printf("THR:\tFRR:\tFAR:\n")
	for _, t := range thresholds {
		fmt.Printf(
			"%.2f\t%.3f\t%.3f\n",
			t,
			genuineStats.RejectionRate(t),
			forgeriesStats.AcceptanceRate(t),
		)
	}
}
