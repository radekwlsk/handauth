package main

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
)

const SplitDefault = 7

var split int

func main() {
	flag.IntVar(&split, "split", SplitDefault, "enroll/test data split point")
	flag.Parse()

	thresholdWeights := flags.ThresholdWeights()

	fmt.Printf(
		"%d\tcols\n"+
			"%d\trows\n"+
			"%d\textract/verify split\n"+
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

	var enrollSamplesIds []int
	var verifySamplesIds []int
	for i := 1; i <= 12; i++ {
		if i < split {
			enrollSamplesIds = append(enrollSamplesIds, i)
		} else {
			verifySamplesIds = append(verifySamplesIds, i)
		}
	}

	if flags.Verbose() {
		fmt.Println("Started")
	}

	users := map[uint8]*cmd.UserFeatures{}
	featuresChan := make(chan *cmd.UserFeatures)

	for i := 1; i <= 100; i++ {
		go cmd.EnrollUserSync(uint8(i), enrollSamplesIds, uint8(*flags.Rows), uint8(*flags.Cols), featuresChan)
	}

	for i := 1; i <= 100; i++ {
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
		fmt.Printf("Enrolled %d users\n", len(users))
	}

	genuineResultsChan := make(chan *cmd.VerificationResult)
	genuineStats := cmd.VerificationStat{
		PositiveCounts: map[float64]uint16{},
		NegativeCounts: map[float64]uint16{},
	}

	for id, user := range users {
		go cmd.VerifyUserSync(id, verifySamplesIds, user, thresholds, thresholdWeights, genuineResultsChan)
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
		fmt.Printf("Verified all genuine users\n")
	}

	forgeriesResultsChan := make(chan *cmd.VerificationResult)
	forgeriesStats := cmd.VerificationStat{
		PositiveCounts: map[float64]uint16{},
		NegativeCounts: map[float64]uint16{},
	}

	count := 0
	for _, user := range users {
		for i := 1; i <= 100; i++ {
			if uint8(i) != user.Id {
				go cmd.VerifyUserSync(
					uint8(i),
					append(enrollSamplesIds, verifySamplesIds...),
					user,
					thresholds,
					thresholdWeights,
					forgeriesResultsChan,
				)
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
