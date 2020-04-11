package commands

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
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

func init() {
	rootCmd.AddCommand(versionCommand)
}

func printSergeDeltaVersion() {
	jww.FEEDBACK.Println("0.1")
}
