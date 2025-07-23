package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Commit    string
	GitLog    string
	BuildTime string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of myapp",
	Long:  `All software has versions. This is myapp's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Git Commit: %s\n", Commit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		//fmt.Printf("Git Log: %s\n", GitLog)

		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
