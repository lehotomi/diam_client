package common

import (
	"log"
	"os"
)

var (
	Log   *log.Logger
	Trace *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
)

//const default_flag = log.Lshortfile | log.Ltime
const default_flag = log.Ltime

func init() {
	Log = log.New(os.Stdout, "", default_flag)
	Trace = log.New(os.Stdout, "TRACE ", default_flag)
	Info = log.New(os.Stdout, "INFO ", default_flag)
	Warn = log.New(os.Stderr, "WARN ", default_flag)
	Error = log.New(os.Stderr, "ERROR ", default_flag)
}
