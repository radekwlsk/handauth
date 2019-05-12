package main

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"io/ioutil"
	"math"
	"strconv"
	"time"
)

const SplitDefault = 0.5

var (
	split float64
	fullResources bool
	start time.Time
)

func main() {
	flag.Float64Var(&split, "split", SplitDefault, "enroll/test data split ratio")
	flag.BoolVar(&fullResources, "full", false, "run test on full dataset")
	flag.Parse()
	cmd.UseFullResources = fullResources
	
	thresholdWeights := flags.ThresholdWeights()

	fmt.Printf(
		"%d\tcols\n"+
			"%d\trows\n"+
			"%.2f\textract/verify split\n"+
			"%v\tthreshold-weights\n",
		*flags.Cols,
		*flags.Rows,
		split,
		thresholdWeights,
	)

	thresholds := flags.Thresholds()
	if flags.Verbose() {
		fmt.Printf("%v\tthresholds\n", thresholds)
	}
	if flags.Verbose() {
		if fullResources {
			fmt.Print("Full test - ")
		}
		fmt.Println("Started")
	}
	
	genuineSamplesUsers := make(map[int][]int)
	forgerySamplesUsers := make(map[[2]int][]int)
	{
		var genuinePath string
		if fullResources {
			genuinePath = cmd.ResourcesFullGenuinePath
		} else {
			genuinePath = cmd.ResourcesTestGenuinePath
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
		var forgeryPath string
		if fullResources {
			forgeryPath = cmd.ResourcesFullForgeryPath
		} else {
			forgeryPath = cmd.ResourcesTestForgeryPath
		}
		files, err = ioutil.ReadDir(forgeryPath)
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
	}
	
	users := map[uint8]*cmd.UserFeatures{}
	{
		if flags.Verbose() { start = time.Now() }
		featuresChan := make(chan *cmd.UserFeatures)
	
		for user, samples := range genuineSamplesUsers {
			enrollSplit := math.Ceil(float64(len(samples)) * split)
			enrollSamples := samples[:int(enrollSplit)]
			go cmd.EnrollUserSync(uint8(user), enrollSamples, uint16(*flags.Rows), uint16(*flags.Cols), featuresChan)
		}
	
		for range genuineSamplesUsers {
			f := <-featuresChan
			if f.Features != nil {
				users[f.Id] = f
				if *flags.VVerbose {
					fmt.Printf("\tEnrolled user %03d\n", f.Id)
				}
			}
		}
	
		close(featuresChan)
		if flags.Verbose() {
			fmt.Printf("Enrolled %d users in %s\n", len(users), time.Since(start))
		}
	}
	
	genuineResultsChan := make(chan *cmd.VerificationResult)
	genuineStats := cmd.VerificationStat{
		PositiveCounts: map[float64]uint16{},
		NegativeCounts: map[float64]uint16{},
	}
	
	{
		if flags.Verbose() { start = time.Now() }
		
		for id, user := range users {
			samples := genuineSamplesUsers[int(id)]
			verifySplit := math.Ceil(float64(len(samples)) * split)
			verifySamples := samples[int(verifySplit):]
			go cmd.VerifyUserSync(id, verifySamples, user, thresholds, thresholdWeights, genuineResultsChan)
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
			if *flags.VVerbose {
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
		if flags.Verbose() {
			fmt.Printf("Verified all genuine users in %s\n", time.Since(start))
		}
	}
	
	forgeriesResultsChan := make(chan *cmd.VerificationResult)
	forgeriesStats := cmd.VerificationStat{
		PositiveCounts: map[float64]uint16{},
		NegativeCounts: map[float64]uint16{},
	}
	
	{
		if flags.Verbose() { start = time.Now() }
		
		for forgerUser, samples := range forgerySamplesUsers {
			go cmd.VerifyUserSync(
				uint8(forgerUser[0]),
				samples,
				users[uint8(forgerUser[1])],
				thresholds,
				thresholdWeights,
				forgeriesResultsChan,
			)
		}
	
		for range forgerySamplesUsers {
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
			if *flags.VVerbose {
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
		if flags.Verbose() {
			fmt.Printf("Verified all forgeries in %s\n", time.Since(start))
		}
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
