package main

import (
	"flag"
	"fmt"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/features"
	"github.com/radekwlsk/handauth/samples"
)

var debug bool

func main() {
	flag.BoolVar(&debug, "debug", false, "show debug steps")
	flag.Parse()

	samples.Debug = debug
	features.Debug = debug

	us, err := cmd.ReadUserSample(1, 1, 1)
	if err != nil {
		panic(err)
	}
	us.Preprocess()
	//sg := samples.NewSampleGrid(us.Sample(), uint8(*flags.Rows), uint8(*flags.Cols))
	//sg.Save("res", "grid", false)

	template := features.NewFeatures(uint8(*flags.Rows), uint8(*flags.Cols), nil)
	template.Extract(us.Sample(), 1)
	fmt.Println(template.GoString())
}
