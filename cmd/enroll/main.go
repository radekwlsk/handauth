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

	us, err := cmd.ReadUserSample(16, 16, 5)
	if err != nil {
		panic(err)
	}
	us.Preprocess()
	sg := samples.NewSampleGrid(us.Sample(), uint16(*flags.Rows), uint16(*flags.Cols))
	//sg.Save("res", "grid", false)
	fmt.Printf("%#v\n", sg)

	template := features.NewFeatures(uint16(*flags.Rows), uint16(*flags.Cols), nil)
	template.Extract(us.Sample(), 1)
	fmt.Println("BEFORE")
	fmt.Println(template.FieldsCount())
	fmt.Println(template.RowsCount())
	fmt.Println(template.ColsCount())
	_ = template.AreaFilter(*flags.FieldAreaThreshold, *flags.RowColAreaThreshold)
	fmt.Println("AFTER")
	fmt.Println(template.FieldsCount())
	fmt.Println(template.RowsCount())
	fmt.Println(template.ColsCount())
}
