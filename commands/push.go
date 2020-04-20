package commands

import (
	"encoding/xml"
	"github.com/dragosv/delta/db"
	"github.com/dragosv/delta/xliff"
	"github.com/jinzhu/gorm"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"path"
	"strconv"
	"time"
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
var database *gorm.DB
var dbJob db.Job
var documentMap map[string]xliff.Document

func init() {
	rootCmd.AddCommand(pushCommand)
}

func runPushCommand() {
	jww.FEEDBACK.Println("push")

	database, err := openDatabase()
	if err != nil {
		panic("failed to connect database")
	}

	defer database.Close()

	database.Where("active = ?", true).First(&dbJob)

	if dbJob.ID != 0 {
		jww.FEEDBACK.Println("active job exists created at " + dbJob.CreatedAt.String())

		return
	}

	dbJob = db.Job{
		CreatedAt: time.Now(),
		Active:    false,
	}

	database.Save(dbJob)

	fs = afero.NewOsFs()
	documentMap = make(map[string]xliff.Document)

	afero.Walk(fs, source, pushWalkFunc)

	for language, document := range documentMap {
		file, err := xml.MarshalIndent(document, "", " ")

		if err != nil {
			panic("failed to write xliff document for language " + language)
		}

		jobID := strconv.FormatUint(uint64(dbJob.ID), 10)
		xliffPath := path.Join(destination, jobID, language+".xliff")

		err = afero.WriteFile(fs, xliffPath, file, 0644)

		if err != nil {
			panic("failed to write xliff file " + xliffPath)
		}
	}
}

func pushWalkFunc(path string, info os.FileInfo, err error) error {
	var data, error = afero.ReadFile(fs, path)

	if error != nil {
		return error
	}

	var document xliff.Document

	document, error = xliff.From(data)

	if error != nil {
		return error
	}

	var dbFile db.File
	var dbTransUnit db.TransUnit
	var dbNote db.Note

	if !document.IsComplete() {
		dbFile = db.File{
			JobID: dbJob.ID,
			Job:   dbJob,
			Path:  path,
		}

		database.Save(dbFile)

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

			database.Save(dbTransUnit)

			for _, xliffNote := range xliffTransUnit.Notes {
				dbNote = db.Note{
					TransUnitID: dbTransUnit.ID,
					TransUnit:   dbTransUnit,
					Data:        xliffNote.Data,
					Language:    xliffNote.Language,
					From:        xliffNote.From,
				}

				database.Save(dbNote)
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
