package commands

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var pushCommand = &cobra.Command{
	Use:   "push",
	Short: "Push command Serge Delta",
	Long:  `Push command Serge Delta.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runPushCommand()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCommand)
}

func runPushCommand() {
	jww.FEEDBACK.Println("push")
}
