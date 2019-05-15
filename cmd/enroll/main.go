package main

import (
	"flag"
	"github.com/radekwlsk/handauth/cmd"
	"github.com/radekwlsk/handauth/samples"
	"github.com/radekwlsk/handauth/signature"
)

var debug bool
var userId int

func main() {
	flag.BoolVar(&debug, "d", false, "show debug steps")
	flag.IntVar(&userId, "u", 1, "user to enroll")
	flag.Parse()

	samples.Debug = debug
	signature.Debug = debug

	//uf := cmd.EnrollUser(uint8(userId), []int{1, 2, 3, 4}, uint16(*flags.Rows), uint16(*flags.Cols))
	//template := uf.Model
	//fmt.Println(template.FieldsCount())
	//fmt.Println(template.RowsCount())
	//fmt.Println(template.ColsCount())
	//
	us, err := cmd.ReadUserSample(16, 16, 5)
	if err != nil {
		panic(err)
	}
	us.Preprocess()
	//us.Sample().Save("res", "test", true)
	//sg := samples.NewSampleGrid(us.Sample(), uint16(*flags.Rows), uint16(*flags.Cols))
	//sg.Save("res", "grid", false)
	//fmt.Printf("%#v\n", sg)
	//template := features.NewModel(uint16(*flags.Rows), uint16(*flags.Cols), nil)
	//template.Extract(us.Sample(), 1)

	//fmt.Println("BEFORE")
	//fmt.Println(template.FieldsCount())
	//fmt.Println(template.RowsCount())
	//fmt.Println(template.ColsCount())
	//_ = template.AreaFilter(*flags.FieldAreaFilter, *flags.RowColAreaFilter)
	//_ = template.StdMeanFilter(*flags.StdMeanFilter)
	//fmt.Println("AFTER")
	//fmt.Println(template.FieldsCount())
	//fmt.Println(template.RowsCount())
	//fmt.Println(template.ColsCount())
}
