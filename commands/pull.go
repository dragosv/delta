package commands

import (
	"errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"path"
	"strconv"
)

var pullCommand = &cobra.Command{
	Use:   "pull",
	Short: "Pull command Delta",
	Long:  `Pull command Delta.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fs = afero.NewOsFs()

		var err error

		database, err = openDatabase(databaseDialect, databaseConnection)
		if err != nil {
			return errors.New("failed to connect database " + err.Error())
		}

		return runPullCommand(source, destination)
	},
}

func init() {
	rootCmd.AddCommand(pullCommand)
}

func runPullCommand(source string, destination string) error {
	jww.FEEDBACK.Println("Running pull")

	database.Where("active = ?", true).First(&dbJob)

	if !database.NewRecord(dbJob.ID) {
		return errors.New("active job does not exists")
	}

	jobID := strconv.FormatUint(uint64(dbJob.ID), 10)

	if plugin != "" {
		job, error := getJob()

		if error != nil {
			return errors.New("failed to get job plugin " + error.Error())
		}

		job.Pull(config, path.Join(destination, jobID))
	}

	return nil
}
