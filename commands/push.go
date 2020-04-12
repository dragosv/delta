package commands

import (
	"github.com/dragosv/delta/xliff"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"os"
)

var pushCommand = &cobra.Command{
	Use:   "push",
	Short: "Push command Delta",
	Long:  `Push command Delta.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runPushCommand()
		return nil
	},
}

var fs afero.Fs
var transUnits []xliff.TransUnit

func init() {
	rootCmd.AddCommand(pushCommand)
}

func runPushCommand() {
	jww.FEEDBACK.Println("push")

	fs = afero.NewOsFs()
	transUnits := []xliff.TransUnit{}

	afero.Walk(fs, source, pushWalkFunc)
}

func pushWalkFunc(path string, info os.FileInfo, err error) error {
	var data, error = afero.ReadFile(fs, path)

	if err != nil {
		return err
	}

	var document xliff.Document

	document, error = xliff.From(data)

	if error != nil {
		return error
	}

	if !document.IsComplete() {
		transUnits = append(transUnits, document.IncompleteTransUnits()...)
	}

	return nil
}
