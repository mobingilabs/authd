package main

import (
	"github.com/golang/glog"
	"github.com/mobingilabs/oath/cmd"
)

func main() {
	glog.CopyStandardLogTo("INFO")
	cmd.Execute()
}
