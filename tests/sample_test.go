package tests

import (
	"flag"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/samples"
	"os"
	"testing"
)

const (
	Creator = 1
	User    = 1
	Index   = 5
)

func BenchmarkPreprocessZhang(b *testing.B) {
	b.SkipNow()
	resFlag := int(cmd.GPDSResources)
	flags.Resources = &resFlag
	testSamples := make([]*samples.Sample, 10)
	for i := range testSamples {
		signature, err := cmd.ReadUserSample(uint16(i+1), uint16(i+1), Index)
		if err != nil {
			panic(err)
		}
		sample := signature.Sample()
		testSamples[i] = sample
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		sig := testSamples[n%10].Copy()
		b.StartTimer()
		sig.Normalize()
		sig.Foreground()
		sig.Crop()
		sig.Resize(samples.TargetWidth, 0.0)
		sig.ZhangSuen()
	}
}

func BenchmarkPreprocessNoneThinning(b *testing.B) {
	b.SkipNow()
	resFlag := int(cmd.GPDSResources)
	flags.Resources = &resFlag
	testSamples := make([]*samples.Sample, 10)
	for i := range testSamples {
		signature, err := cmd.ReadUserSample(uint16(i+1), uint16(i+1), Index)
		if err != nil {
			panic(err)
		}
		sample := signature.Sample()
		testSamples[i] = sample
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		sig := testSamples[n%10].Copy()
		b.StartTimer()
		sig.Normalize()
		sig.Foreground()
		sig.Crop()
		sig.Resize(samples.TargetWidth, 0.0)
	}
}

func BenchmarkZhangThinning(b *testing.B) {
	b.SkipNow()
	resFlag := int(cmd.GPDSResources)
	flags.Resources = &resFlag
	testSamples := make([]*samples.Sample, 10)
	for i := range testSamples {
		signature, err := cmd.ReadUserSample(uint16(i+1), uint16(i+1), Index)
		if err != nil {
			panic(err)
		}
		sample := signature.Sample()
		sample.Normalize()
		sample.Foreground()
		sample.Crop()
		sample.Resize(samples.TargetWidth, 0.0)
		testSamples[i] = sample
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		sig := testSamples[n%10].Copy()
		b.StartTimer()
		sig.ZhangSuen()
	}
}

func BenchmarkEnroll(b *testing.B) {
	b.SkipNow()
	resFlag := int(cmd.GPDSResources)
	flags.Resources = &resFlag
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cmd.EnrollUser(uint16((n%10)+1), []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 12, 60)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
