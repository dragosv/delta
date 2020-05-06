package commands

import (
	"encoding/xml"
	"errors"
	"github.com/dragosv/delta/db"
	"github.com/dragosv/delta/xliff"
	"github.com/jinzhu/gorm"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"path"
	"strconv"
)

var pushCommand = &cobra.Command{
	Use:   "push",
	Short: "Push command Delta",
	Long:  `Push command Delta.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fs = afero.NewOsFs()

		var err error

		database, err = openDatabase(databaseDialect, databaseConnection)
		if err != nil {
			return errors.New("failed to connect database " + err.Error())
		}

		return runPushCommand(source, destination)
	},
}

var fs afero.Fs
var database *gorm.DB
var dbJob db.Job
var sourceDocumentMap map[string]xliff.Document
var documentMap map[string]xliff.Document

func init() {
	rootCmd.AddCommand(pushCommand)
}

func runPushCommand(source string, destination string) error {
	jww.FEEDBACK.Println("Running push...")

	database.Where("active = ?", true).First(&dbJob)

	if !database.NewRecord(dbJob.ID) {
		return errors.New("active job exists created at " + dbJob.CreatedAt.String())
	}

	dbJob = db.Job{
		Active: true,
	}

	err := database.Create(&dbJob).Error

	if err != nil {
		return err
	}

	sourceDocumentMap = make(map[string]xliff.Document)
	documentMap = make(map[string]xliff.Document)

	afero.Walk(fs, source, pushWalkFunc)

	for path, document := range sourceDocumentMap {
		processErr := processSourceDocument(path, document)

		if processErr != nil {
			panic(processErr)
		}
	}

	jobID := strconv.FormatUint(uint64(dbJob.ID), 10)

	for language, document := range documentMap {
		file, err := xml.MarshalIndent(document, "", " ")

		if err != nil {
			return errors.New("failed to write xliff document for language " + language)
		}

		xliffPath := path.Join(destination, jobID, language+".xliff")

		err = afero.WriteFile(fs, xliffPath, file, 0644)

		if err != nil {
			return errors.New("failed to write xliff file " + xliffPath)
		}
	}

	if plugin != "" {
		job, error := getJob()

		if error != nil {
			return errors.New("failed to get job plugin " + error.Error())
		}

		job.Push(config, path.Join(destination, jobID))
	}

	return nil
}

func pushWalkFunc(path string, info os.FileInfo, err error) error {
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

	sourceDocumentMap[path] = document

	return nil
}

func processSourceDocument(path string, document xliff.Document) error {
	var dbFile db.File
	var dbTransUnit db.TransUnit
	var dbNote db.Note

	if !document.IsComplete() {
		dbFile = db.File{
			JobID: dbJob.ID,
			Job:   dbJob,
			Path:  path,
		}

		err := database.Create(&dbFile).Error

		if err != nil {
			return err
		}

		var incompleteTransUnits = document.IncompleteTransUnits()
		for _, xliffTransUnit := range incompleteTransUnits {
			dbTransUnit = db.TransUnit{
				Resname:        xliffTransUnit.Resname,
				Path:           path,
				Qualifier:      xliffTransUnit.ID,
				State:          xliffTransUnit.Target.State,
				StateQualifier: xliffTransUnit.Target.StateQualifier,
				Source:         xliffTransUnit.Source.Data,
				Target:         xliffTransUnit.Target.Data,
				SourceLanguage: xliffTransUnit.Source.Language,
				TargetLanguage: xliffTransUnit.Target.Language,
			}

			err = database.Create(&dbTransUnit).Error

			if err != nil {
				return err
			}

			for _, xliffNote := range xliffTransUnit.Notes {
				dbNote = db.Note{
					TransUnitID: dbTransUnit.ID,
					TransUnit:   dbTransUnit,
					Data:        xliffNote.Data,
					Language:    xliffNote.Language,
					From:        xliffNote.From,
				}

				err = database.Create(&dbNote).Error

				if err != nil {
					return err
				}
			}

			var transUnit = xliffTransUnit
			transUnit.ID = strconv.FormatUint(uint64(dbTransUnit.ID), 10)

			document := documentMap[xliffTransUnit.Target.Language]

			if document.Version == "" {
				document.Version = "1.2"
			}

			if len(document.Files) == 0 {
				document.Files = append(document.Files, xliff.File{
					Original:       xliffTransUnit.Target.Language + ".xliff",
					SourceLanguage: xliffTransUnit.Source.Language,
					Datatype:       "plaintext",
					TargetLanguage: xliffTransUnit.Target.Language,
					Header:         xliff.Header{Tool: xliff.Tool{ToolID: "delta", ToolName: "delta", ToolVersion: "0.1", BuildNum: "0"}},
					Body:           xliff.Body{},
				})
			}

			document.Files[0].Body.TransUnits = append(document.Files[0].Body.TransUnits, transUnit)

			documentMap[xliffTransUnit.Target.Language] = document
		}
	}

	return nil
}
