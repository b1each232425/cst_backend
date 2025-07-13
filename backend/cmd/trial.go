/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"w2w.io/cmn"
)

// trialCmd represents the trial command
var trialCmd = &cobra.Command{
	Use:   "trial",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: trial,
}

func init() {
	rootCmd.AddCommand(trialCmd)
}

func trial(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("input trial params")
		return
	}

	for _, v := range args {
		fn := cmn.B64UEncode([]byte(v))
		s, err := cmn.B64UDecode(fn)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		fmt.Printf(`
 origin: %s
decoded: %s
encoded: %s
`, v, s, fn)
	}
}
