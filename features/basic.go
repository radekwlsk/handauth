package features

import (
	"github.com/radekwlsk/handauth/samples"
	"gonum.org/v1/gonum/stat"
	"math"
	"sync"
)

type BasicFeatures map[string]*Feature

func NewBasicFeatures() *BasicFeatures {
	return &BasicFeatures{
		"aspect":   NewFeature("aspect", aspect),
		"length":   NewFeature("length", length),
		"gradient": NewFeature("gradient", gradient),
	}
}

func (f BasicFeatures) Score(filename string) (float64, *BasicFeatures) {
	pattern := NewBasicFeatures()
	sample := samples.NewSample(filename)
	sample.Preprocess(f["aspect"].mean)
	pattern.Extract(sample, 1)

	scores := make([]float64, 0)
	for key := range f {
		//fmt.Printf("%s: pattern: %f, mean: %f, std: %f\n", key, pattern[key].mean, f[key].mean, f[key].std)
		score := stat.StdScore((*pattern)[key].mean, f[key].mean, f[key].std)
		scores = append(scores, math.Abs(score))
	}
	return stat.Mean(scores, nil), pattern
}

func (f BasicFeatures) Extract(sample *samples.Sample, nSamples int) {
	sample.Update()

	var wg sync.WaitGroup
	wg.Add(len(f))

	for _, feature := range f {
		go func(f *Feature) {
			defer wg.Done()
			f.Update(sample, nSamples)
		}(feature)
	}

	wg.Wait()
}
