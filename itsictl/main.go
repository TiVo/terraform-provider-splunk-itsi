package main

import (
	"fmt"
	"runtime"

	"github.com/tivo/terraform-provider-splunk-itsi/itsictl/cmd"
)

var (
	// These variables are set in the build step
	version   string
	commit    string
	buildTime string
)

func printInfo() {
	fmt.Printf("itsictl %s %s/%s (%s #%s)\n", version, runtime.GOOS, runtime.GOARCH, buildTime, commit)
}

func main() {
	printInfo()
	cmd.Execute()
}
