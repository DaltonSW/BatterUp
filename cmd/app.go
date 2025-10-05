package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = cobra.Command{
	Use:   "batterup",
	Short: "Monitor MLB games in your terminal",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Howdy!")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
