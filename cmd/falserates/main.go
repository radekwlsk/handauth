package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/signature"
	"github.com/radekwlsk/handauth/signature/features"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const SplitDefault = 0.5
const TestStartTimeFormat = "20060201-150405"

var (
	split            float64
	thresholds       []float64
	thresholdWeights map[signature.AreaType]float64
	fullResources    bool
	start            time.Time
	startString      string
	outFileName      string
	outWriter        *csv.Writer
	configWriter     *csv.Writer
	workingDir       string
	testMessage      string
)

func configRecords() [][]string {
	var dataset string
	switch cmd.ResourceType(*flags.Resources) {
	case cmd.GPDSResources:
		dataset = fmt.Sprintf("GPDS%d", *flags.GPDSUsers)
		break
	case cmd.SigCompResources:
		var f string
		if fullResources {
			f = "Full"
		} else {
			f = "Test"
		}
		dataset = fmt.Sprintf("SigComp%s", f)
	case cmd.MCYTResources:
		dataset = fmt.Sprintf("MCYT")
		break
	default:
		log.Fatalf("no such dataset: %d", cmd.ResourceType(*flags.Resources))
	}
	config := [][]string{
		{"message", testMessage},
		{"date", start.String()},
		{"full data", fmt.Sprintf("%v", fullResources)},
		{"dataset", dataset},
		{"cols", fmt.Sprintf("%d", *flags.Cols)},
		{"rows", fmt.Sprintf("%d", *flags.Rows)},
		{"split", fmt.Sprintf("%.2f", split)},
		{"using area filter", fmt.Sprintf("%v", !*flags.AreaFilterOff)},
		{"field area min threshold", fmt.Sprintf("%.3f", *flags.AreaFilterFieldThreshold)},
		{"row/col area min threshold", fmt.Sprintf("%.3f", *flags.AreaFilterRowColThreshold)},
		{"using std-mean filter", fmt.Sprintf("%v", !*flags.StdMeanFilterOff)},
		{"std-mean max mean ratio threshold", fmt.Sprintf("%.3f", *flags.StdMeanFilterThreshold)},
	}
	for a, w := range thresholdWeights {
		config = append(config, []string{fmt.Sprintf("%s weight", a), fmt.Sprintf("%.2f", w)})
	}
	for t, f := range signature.AreaFlags {
		config = append(config, []string{fmt.Sprintf("using %s", t), fmt.Sprintf("%v", f)})
	}
	for t, f := range features.FeatureFlags {
		config = append(config, []string{fmt.Sprintf("using %s", t), fmt.Sprintf("%v", f)})
	}
	return config
}

func main() {
	flag.Float64Var(&split, "split", SplitDefault, "enroll/test data split ratio")
	flag.BoolVar(&fullResources, "full", false, "run test on full SigComp dataset if flag res = 0")
	flag.StringVar(&outFileName, "o", "out.csv", "output file")
	flag.StringVar(&testMessage, "m", "", "message to be associated with a test")
	flag.Parse()
	cmd.UseFullResources = fullResources

	start = time.Now()
	startString = start.Format(TestStartTimeFormat)

	if flags.Verbose() {
		log.Println("Starting")
	}

	{
		workingDir, _ = os.Getwd()
		if !strings.HasSuffix(outFileName, ".csv") {
			outFileName += ".csv"
		}
		file, err := os.Create(path.Join(workingDir, "res", startString+"_"+outFileName))
		if err != nil {
			panic(err)
		}
		defer file.Close()

		outWriter = csv.NewWriter(file)
		defer outWriter.Flush()
	}

	thresholdWeights = flags.ThresholdWeights()
	thresholds = flags.Thresholds()

	{
		ext := filepath.Ext(outFileName)
		configFileName := startString + "_" + strings.TrimSuffix(outFileName, ext) + "_config" + ext
		file, err := os.Create(path.Join(workingDir, "res", configFileName))
		if err != nil {
			panic(err)
		}
		defer file.Close()

		configWriter = csv.NewWriter(file)
		defer configWriter.Flush()
		_ = configWriter.WriteAll(configRecords())
		configWriter.Flush()
	}

	genuineSamplesUsers := cmd.GenuineUsers(fullResources)
	forgerySamplesUsers := cmd.ForgeryUsers(fullResources)

	users := map[uint16]*signature.UserModel{}
	{
		start := time.Now()
		featuresChan := make(chan *signature.UserModel)

		for user, samples := range genuineSamplesUsers {
			enrollSplit := math.Ceil(float64(len(samples)) * split)
			enrollSamples := samples[:int(enrollSplit)]
			go cmd.EnrollUserSync(uint16(user), enrollSamples, uint16(*flags.Rows), uint16(*flags.Cols), featuresChan)
		}

		for range genuineSamplesUsers {
			f := <-featuresChan
			if f.Model != nil {
				users[f.Id] = f
				if *flags.VVerbose {
					log.Printf("\tEnrolled user %03d\n", f.Id)
				}
			}
		}
		close(featuresChan)

		elapsed := time.Since(start)
		_ = configWriter.Write([]string{"enroll duration", elapsed.String()})
		if flags.Verbose() {
			log.Printf("Enrolled %d users in %s\n", len(users), elapsed)
		}
	}

	genuineResultsChan := make(chan *cmd.VerificationResult)
	genuineStats := cmd.VerificationStat{
		PositiveCounts: map[float64]uint16{},
		NegativeCounts: map[float64]uint16{},
	}

	{
		start := time.Now()

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
				log.Printf("\tVerified user %03d\n", r.TemplateUserId)
				for i, t := range thresholds {
					log.Printf(
						"\t\t%.2f: %d/%d\n",
						t,
						r.SuccessCounts[i],
						r.RejectedCounts[i],
					)
				}
			}
		}
		close(genuineResultsChan)

		elapsed := time.Since(start)
		_ = configWriter.Write([]string{"genuine verification duration", elapsed.String()})
		if flags.Verbose() {
			log.Printf("Verified all genuine users in %s\n", elapsed)
		}
	}

	forgeriesResultsChan := make(chan *cmd.VerificationResult)
	forgeriesStats := cmd.VerificationStat{
		PositiveCounts: map[float64]uint16{},
		NegativeCounts: map[float64]uint16{},
	}

	{
		start := time.Now()

		for forgerUser, samples := range forgerySamplesUsers {
			go cmd.VerifyUserSync(
				uint16(forgerUser[0]),
				samples,
				users[uint16(forgerUser[1])],
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
				log.Printf(
					"\tVerified user %03d as %03d\n",
					r.SampleUserId,
					r.TemplateUserId,
				)
				for i, t := range thresholds {
					log.Printf(
						"\t\t%.2f: %d/%d\n",
						t,
						r.SuccessCounts[i],
						r.RejectedCounts[i],
					)
				}
			}
		}
		close(forgeriesResultsChan)

		elapsed := time.Since(start)
		_ = configWriter.Write([]string{"forgeries verification duration", elapsed.String()})
		if flags.Verbose() {
			log.Printf("Verified all forgeries in %s\n", elapsed)
		}
	}
	_ = outWriter.Write([]string{"threshold", "frr", "far"})
	for _, t := range thresholds {
		_ = outWriter.Write([]string{
			fmt.Sprintf("%.2f", t),
			fmt.Sprintf("%.4f", genuineStats.RejectionRate(t)),
			fmt.Sprintf("%.4f", forgeriesStats.AcceptanceRate(t)),
		})
	}
	_ = configWriter.Write([]string{"total test duration", time.Since(start).String()})
}
