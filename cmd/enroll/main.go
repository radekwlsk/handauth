package main

import (
	"flag"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"image"
	"image/color"
	"log"
)

var debug bool

//var resPath string

type mask struct {
	img image.Image
}

func (m *mask) ColorModel() color.Model {
	return color.AlphaModel
}

func (m *mask) Bounds() image.Rectangle {
	return m.img.Bounds()
}

var black = color.Gray{Y: 0xff}

func (m *mask) At(x, y int) color.Color {
	if m.img.At(x, y) == black {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

func drawDot(dst *image.RGBA, center image.Point, c color.RGBA) {
	for _, x := range []int{center.X - 1, center.X, center.X + 1} {
		for _, y := range []int{center.Y - 1, center.Y, center.Y + 1} {
			p := image.Point{X: x, Y: y}
			if p.In(dst.Bounds()) {
				dst.SetRGBA(p.X, p.Y, c)
			}
		}
	}
}

func drawLines(dst *image.RGBA, center image.Point, c color.RGBA) {
	for x := 0; x < dst.Rect.Max.X; x++ {
		p := image.Point{X: x, Y: center.Y}
		if p.In(dst.Bounds()) {
			dst.SetRGBA(p.X, p.Y, c)
		}
	}
	for y := 0; y < dst.Rect.Max.X; y++ {
		p := image.Point{X: center.X, Y: y}
		if p.In(dst.Bounds()) {
			dst.SetRGBA(p.X, p.Y, c)
		}
	}
}

var blackRGBA = color.RGBA{A: 255}
var redRGBA = color.RGBA{R: 255, A: 255}

func main() {
	flag.BoolVar(&debug, "d", false, "show debug steps")
	//flag.StringVar(&resPath, "path", ".", "path of dir with resources")
	flag.Parse()

	//user := 25
	//sampleId := 9

	var beforeCount, midCount, afterCount uint64

	t := true
	flags.AreaFilterOff = &t
	flags.StdFilterOff = &t

	for u := 1; u < 31; u++ {
		um := cmd.EnrollUser(uint16(u), []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 12, 60)
		beforeCount += um.Model.FeaturesCount()
		if err := um.Model.AreaFilter(flags.AreaFilterFieldThresholdDefault,
			flags.AreaFilterRowColThresholdDefault); err != nil {
			panic(err)
		}
		midCount += um.Model.FeaturesCount()
		if err := um.Model.StdFilter(flags.StdFilterThresholdDefault); err != nil {
			panic(err)
		}
		afterCount += um.Model.FeaturesCount()
		log.Print(u)
	}

	log.Printf("before avg: %d", beforeCount/30)
	log.Printf("area avg: %d", midCount/30)
	log.Printf("std avg: %d", afterCount/30)

	//sample, err := cmd.ReadUserSample(uint16(user), uint16(user), uint8(sampleId))
	//if err != nil {
	//	log.Fatalf(err.Error())
	//}
	////sample.Sample().Save("res", "thinning-before", false)
	//sample.Preprocess()
	//
	//sg := samples.NewSampleGrid(sample.Sample(), 15, 30)
	//for c := 0; c < 30;  c++ {
	//	for r := 0; r < 15;  r++ {
	//		sg.At(r, c).Save("res/grid", fmt.Sprintf("c%dr%d", c, r), false)
	//	}
	//}
	//um := signature.NewModel(15, 30, nil)
	//um.Extract(sample.Sample(), 1)
	//for r := 0; r < 15;  r++ {
	//	for c := 0; c < 30;  c++ {
	//		hog := um.Grid(r, c)[features.HOGFeatureType].Value()
	//		fmt.Printf("%d\t%d\t%f\n", r, c, hog)
	//	}
	//}

	//sampleMat := sample.Sample().Mat()
	//img, err := sampleMat.ToImage()
	//if err != nil {
	//	panic(err)
	//}
	//dst := image.NewRGBA(img.Bounds())
	//draw.Draw(dst, img.Bounds(), img, image.ZP, draw.Src)
	//mc := sample.Sample().CenterOfMass()
	//name := fmt.Sprintf("%d_%d_mcx-%d_mcy-%d",user, sampleId, mc.X, mc.Y)
	////drawDot(dst, mc, redRGBA)
	//drawLines(dst, mc, redRGBA)
	//mergedFile, err := os.Create(path.Join("res", name + ".png"))
	//_ = png.Encode(mergedFile, dst)
	////sample.Sample().Save("res", "thinning-final", false)

	//files, err := ioutil.ReadDir(resPath)
	//if err != nil {
	//	log.Fatalf(err.Error())
	//}
	//for _, f := range files {
	//	if !f.IsDir() {
	//		filePath := path.Join(resPath, f.Name())
	//		sample, err := samples.NewUserSample(f.Name(), filePath)
	//		if err != nil {
	//			log.Fatalf(err.Error())
	//		}
	//		sample.Preprocess()
	//		//sample.Sample().Save(resPath, "processed-"+f.Name() , false)
	//
	//		sobelX := gocv.NewMat()
	//		sobelY := gocv.NewMat()
	//		gocv.SpatialGradient(sample.Sample().Mat(), &sobelX, &sobelY, 3, gocv.BorderReplicate)
	//		sobelX.ConvertTo(&sobelX, gocv.MatTypeCV8UC1)
	//		sobelY.ConvertTo(&sobelY, gocv.MatTypeCV8UC1)
	//
	//
	//		template := signature.NewModel(uint16(*flags.Rows), uint16(*flags.Cols), nil)
	//		template.Extract(sample.Sample(), 1)
	//		sample.Close()
	//		log.Println(f.Name())
	//		log.Println(template.Basic())
	//		log.Printf("sobel X: %d", gocv.CountNonZero(sobelX))
	//		log.Printf("sobel Y: %d", gocv.CountNonZero(sobelY))
	//		sobelX.Close()
	//		sobelY.Close()
	//	}
	//}
}
