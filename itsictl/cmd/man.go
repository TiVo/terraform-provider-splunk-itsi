package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// Add a hidden command to generate manpages
var manCmd = &cobra.Command{
	Use:    "genman",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		manDir := "./man/"
		header := &doc.GenManHeader{
			Title:   ITSICTL,
			Section: "1",
		}
		os.MkdirAll("./man", os.ModePerm)
		return doc.GenManTree(rootCmd, header, manDir)
	},
}

func init() {
	rootCmd.AddCommand(manCmd)
}
