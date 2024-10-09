package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// These variables are set in the build step
	version   string
	commit    string
	buildTime string
)

func printVersion() {
	fmt.Printf("itsictl %s %s/%s (%s #%s)\n", version, runtime.GOOS, runtime.GOARCH, buildTime, commit)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
