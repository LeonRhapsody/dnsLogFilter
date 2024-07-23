package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "logTransfer",
	Short: "DNS Log Filter Program",
	Long:  `DNS Log Filter Program,support multiple rules`,
	Run: func(cmd *cobra.Command, args []string) {
		// 默认行为
		fmt.Println("No command provided, executing default action.")
		// 这里可以放置你想要执行的默认操作
	},
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// You can define persistent flags here and bind them if necessary
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.myapp.yaml)")
}
