package main

import (
	"flag"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/cmd/flags"
	"github.com/radekwlsk/handauth/samples"
)

func main() {
	//samples.Debug = true
	flag.Parse()
	us, err := cmd.ReadUserSample(1, 1, 1)
	if err != nil {
		panic(err)
	}
	us.Preprocess()
	sg := samples.NewSampleGrid(us.Sample(), uint8(*flags.Rows), uint8(*flags.Cols))
	sg.Save("res", "grid", false)
}
