package tests

import (
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/signature"
	"testing"
)

var ResFlag = int(cmd.GPDSResources)
var GPDSFlag = 4000

func BenchmarkGPDSEnroll2x6(b *testing.B)   { benchmarkGPDSEnroll(nil, 2, 6, b) }
func BenchmarkGPDSEnroll5x15(b *testing.B)  { benchmarkGPDSEnroll(nil, 5, 15, b) }
func BenchmarkGPDSEnroll10x30(b *testing.B) { benchmarkGPDSEnroll(nil, 10, 30, b) }
func BenchmarkGPDSEnroll10x60(b *testing.B) { benchmarkGPDSEnroll(nil, 10, 60, b) }
func BenchmarkGPDSEnroll20x30(b *testing.B) { benchmarkGPDSEnroll(nil, 20, 30, b) }
func BenchmarkGPDSEnroll20x60(b *testing.B) { benchmarkGPDSEnroll(nil, 20, 60, b) }

func BenchmarkGPDSVerify2x6(b *testing.B)   { benchmarkGPDSVerify(nil, 2, 6, b) }
func BenchmarkGPDSVerify5x15(b *testing.B)  { benchmarkGPDSVerify(nil, 5, 15, b) }
func BenchmarkGPDSVerify10x30(b *testing.B) { benchmarkGPDSVerify(nil, 10, 30, b) }
func BenchmarkGPDSVerify10x60(b *testing.B) { benchmarkGPDSVerify(nil, 10, 60, b) }
func BenchmarkGPDSVerify20x30(b *testing.B) { benchmarkGPDSVerify(nil, 20, 30, b) }
func BenchmarkGPDSVerify20x60(b *testing.B) { benchmarkGPDSVerify(nil, 20, 60, b) }

func BenchmarkGPDSEnrollAll(b *testing.B) {
	benchmarkGPDSEnroll(map[signature.AreaType]bool{
		signature.BasicAreaType: true,
		signature.RowAreaType:   true,
		signature.ColAreaType:   true,
		signature.GridAreaType:  true,
	}, 10, 30, b)
}
func BenchmarkGPDSEnrollBasic(b *testing.B) {
	benchmarkGPDSEnroll(map[signature.AreaType]bool{
		signature.BasicAreaType: true,
		signature.RowAreaType:   false,
		signature.ColAreaType:   false,
		signature.GridAreaType:  false,
	}, 10, 30, b)
}
func BenchmarkGPDSEnrollGrid(b *testing.B) {
	benchmarkGPDSEnroll(map[signature.AreaType]bool{
		signature.BasicAreaType: false,
		signature.RowAreaType:   false,
		signature.ColAreaType:   false,
		signature.GridAreaType:  true,
	}, 10, 30, b)
}
func BenchmarkGPDSEnrollRC(b *testing.B) {
	benchmarkGPDSEnroll(map[signature.AreaType]bool{
		signature.BasicAreaType: false,
		signature.RowAreaType:   true,
		signature.ColAreaType:   true,
		signature.GridAreaType:  false,
	}, 10, 30, b)
}

func BenchmarkGPDSVerifyAll(b *testing.B) {
	benchmarkGPDSVerify(map[signature.AreaType]bool{
		signature.BasicAreaType: true,
		signature.RowAreaType:   true,
		signature.ColAreaType:   true,
		signature.GridAreaType:  true,
	}, 10, 30, b)
}
func BenchmarkGPDSVerifyBasic(b *testing.B) {
	benchmarkGPDSVerify(map[signature.AreaType]bool{
		signature.BasicAreaType: true,
		signature.RowAreaType:   false,
		signature.ColAreaType:   false,
		signature.GridAreaType:  false,
	}, 10, 30, b)
}
func BenchmarkGPDSVerifyGrid(b *testing.B) {
	benchmarkGPDSVerify(map[signature.AreaType]bool{
		signature.BasicAreaType: false,
		signature.RowAreaType:   false,
		signature.ColAreaType:   false,
		signature.GridAreaType:  true,
	}, 10, 30, b)
}
func BenchmarkGPDSVerifyRC(b *testing.B) {
	benchmarkGPDSVerify(map[signature.AreaType]bool{
		signature.BasicAreaType: false,
		signature.RowAreaType:   true,
		signature.ColAreaType:   true,
		signature.GridAreaType:  false,
	}, 10, 30, b)
}

func benchmarkGPDSEnroll(area map[signature.AreaType]bool, rows, cols uint16, b *testing.B) {
	flags.Resources = &ResFlag
	flags.GPDSUsers = &GPDSFlag
	if area != nil {
		signature.AreaFlags = area
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		userId := uint16(n%4000 + 1)
		cmd.EnrollUser(userId, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, rows, cols)
	}
}

func benchmarkGPDSVerify(area map[signature.AreaType]bool, rows, cols uint16, b *testing.B) {
	flags.Resources = &ResFlag
	flags.GPDSUsers = &GPDSFlag
	if area != nil {
		signature.AreaFlags = area
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		userId := uint16(n%4000 + 1)
		sampleId := uint8(n%14 + 11)
		um := cmd.EnrollUser(userId, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, rows, cols)
		b.StartTimer()
		sample, _ := cmd.ReadUserSample(userId, userId, sampleId)
		sample.Preprocess()
		score, _ := um.Model.Score(sample.Sample())
		sample.Close()
		_, _ = score.Check(1.25, nil)
	}
}
