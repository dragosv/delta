package commands

import (
	"errors"
	"github.com/dragosv/delta/db"
	"github.com/dragosv/delta/xliff"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"path"
	"strconv"
)

var destinationDocumentMap map[string]xliff.Document

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

	destinationDocumentMap = make(map[string]xliff.Document)

	afero.Walk(fs, path.Join(destination, jobID), pullWalkFunc)

	for path, document := range destinationDocumentMap {
		processErr := processDestinationDocument(path, document)

		if processErr != nil {
			panic(processErr)
		}
	}

	sourceDocumentMap = make(map[string]xliff.Document)

	afero.Walk(fs, source, sourceWalkFunc)

	for path, document := range sourceDocumentMap {
		processErr := writeSourceDocument(path, document)

		if processErr != nil {
			panic(processErr)
		}
	}

	dbJob.Active = false

	return database.Save(&dbJob).Error
}

func writeSourceDocument(path string, document xliff.Document) error {
	var dbFile db.File
	var dbTransUnit db.TransUnit
	var write bool
	var newDocument xliff.Document
	var newFile xliff.File

	newDocument = xliff.Document{
		Version: document.Version,
		Files:   []xliff.File{},
	}

	for _, file := range document.Files {
		newFile = xliff.File{
			Original:       file.Original,
			SourceLanguage: file.SourceLanguage,
			Datatype:       file.Datatype,
			TargetLanguage: file.TargetLanguage,
			Header:         file.Header,
			Body:           xliff.Body{TransUnits: []xliff.TransUnit{}},
		}

		for _, transUnit := range file.Body.TransUnits {
			if !transUnit.IsComplete() {
				if database.NewRecord(dbFile) {
					database.Where("job_id = ? and language = ?", dbJob.ID, document.Files[0].TargetLanguage).First(&dbFile)
				}

				if !database.NewRecord(dbFile) {
					database.Where("file_id = ? and qualifier = ?", dbFile.ID, transUnit.ID).First(&dbTransUnit)

					if !database.NewRecord(dbTransUnit) {
						transUnit.Target.Data = dbTransUnit.Target
						transUnit.Target.State = dbTransUnit.State
						transUnit.Target.StateQualifier = dbTransUnit.StateQualifier

						write = true
					}
				}
			}

			newFile.Body.TransUnits = append(newFile.Body.TransUnits, transUnit)
		}

		newDocument.Files = append(newDocument.Files, newFile)
	}

	if write {
		return writeDocument(newDocument, path)
	}

	return nil
}

func processDestinationDocument(path string, document xliff.Document) error {
	var dbFile db.File
	var dbTransUnit db.TransUnit

	if len(document.Files) == 0 {
		return nil
	}

	for _, file := range document.Files {
		database.Where("job_id = ? and language = ?", dbJob.ID, document.Files[0].TargetLanguage).First(&dbFile)

		if !database.NewRecord(dbFile) {
			for _, transUnit := range file.Body.TransUnits {
				database.Where("file_id = ? and identifier = ?", dbFile.ID, transUnit.ID).First(&dbTransUnit)

				if !database.NewRecord(dbTransUnit) {
					dbTransUnit.Target = transUnit.Target.Data
					dbTransUnit.State = transUnit.Target.State
					dbTransUnit.StateQualifier = transUnit.Target.StateQualifier

					err := database.Save(&dbTransUnit).Error

					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func pullWalkFunc(path string, info os.FileInfo, err error) error {
	if info == nil {
		return nil
	}

	if info.IsDir() {
		return nil
	}

	var data, error = afero.ReadFile(fs, path)

	if error != nil {
		return error
	}

	var document xliff.Document

	document, error = xliff.From(data)

	if error != nil {
		return error
	}

	destinationDocumentMap[path] = document

	return nil
}
