package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"os"
)

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Serge Delta",
	Long:  `All software has versions. This is Serge Delta's.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printSergeDeltaVersion()
		return nil
	},
}

func Execute() {
	if err := versionCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func printSergeDeltaVersion() {
	jww.FEEDBACK.Println("0.1")
}
