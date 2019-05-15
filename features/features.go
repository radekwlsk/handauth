package features

import (
	"log"
	"os"
)

var Debug = false
var logger = log.New(os.Stdout, "[features] ", log.Lshortfile+log.Ltime)
