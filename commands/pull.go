package commands

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var pullCommand = &cobra.Command{
	Use:   "pull",
	Short: "Pull command Delta",
	Long:  `Pull command Delta.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runPullCommand()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullCommand)
}

func runPullCommand() {
	jww.FEEDBACK.Println("pull")
}
