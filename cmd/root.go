package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
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
		log.Fatal(err)
		return
	}
}

func init() {
	// You can define persistent flags here and bind them if necessary
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.myapp.yaml)")
}
