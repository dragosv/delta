package commands

import (
	"errors"
	"github.com/dragosv/delta/db"
	"github.com/dragosv/delta/xliff"
	guuid "github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
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
var patternRegexp *regexp.Regexp

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

	patternRegexp, err = regexp.Compile(languagePattern)

	if err != nil {
		return err
	}

	sourceDocumentMap = make(map[string]xliff.Document)
	documentMap = make(map[string]xliff.Document)

	afero.Walk(fs, source, sourceWalkFunc)

	for path, document := range sourceDocumentMap {
		processErr := processSourceDocument(path, document)

		if processErr != nil {
			panic(processErr)
		}
	}

	jobID := strconv.FormatUint(uint64(dbJob.ID), 10)

	for language, document := range documentMap {
		xliffPath := path.Join(destination, jobID, language+".xliff")

		writeDocument(document, xliffPath)
	}

	if plugin != "" {
		job, error := getJob()

		if error != nil {
			return errors.New("failed to get job plugin " + error.Error())
		}

		return job.Push(config, path.Join(destination, jobID))
	}

	return nil
}

func sourceWalkFunc(path string, info os.FileInfo, err error) error {
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
	var dbIdentifier db.Identifier

	if !document.IsComplete() {
		indexes := patternRegexp.FindStringSubmatchIndex(path)

		if indexes == nil {
			return errors.New("No language could be identified for file " + path)
		}

		start := len(indexes) - 2
		end := len(indexes) - 1
		mainPath := replaceAtIndex(path, sourceLanguage, indexes[start], indexes[end])

		database.Where("job_id = ? and path = ?", dbJob.ID, document.Files[0].TargetLanguage).First(&dbFile)

		if database.NewRecord(dbFile) {
			dbFile = db.File{
				JobID:    dbJob.ID,
				Job:      dbJob,
				Path:     path,
				Language: document.Files[0].TargetLanguage,
			}

			err := database.Create(&dbFile).Error

			if err != nil {
				return err
			}
		}

		var incompleteTransUnits = document.IncompleteTransUnits()
		for _, xliffTransUnit := range incompleteTransUnits {
			var identifier string

			database.Where("job_id = ? and path = ? and qualifier = ?", dbJob.ID, mainPath, xliffTransUnit.ID).First(&dbIdentifier)

			if database.NewRecord(dbIdentifier) {
				identifier = strings.Replace(guuid.New().String(), "-", "", -1)

				dbIdentifier = db.Identifier{
					Model:     gorm.Model{},
					JobID:     dbJob.ID,
					Job:       dbJob,
					Data:      identifier,
					Qualifier: xliffTransUnit.ID,
					Path:      mainPath,
				}

				err := database.Create(&dbIdentifier).Error

				if err != nil {
					return err
				}
			} else {
				identifier = dbIdentifier.Data
			}

			dbTransUnit = db.TransUnit{
				Resname:        xliffTransUnit.Resname,
				Path:           path,
				Identifier:     identifier,
				Qualifier:      xliffTransUnit.ID,
				State:          xliffTransUnit.Target.State,
				StateQualifier: xliffTransUnit.Target.StateQualifier,
				Source:         xliffTransUnit.Source.Data,
				Target:         xliffTransUnit.Target.Data,
				SourceLanguage: xliffTransUnit.Source.Language,
				TargetLanguage: xliffTransUnit.Target.Language,
				FileID:         dbFile.ID,
			}

			err := database.Create(&dbTransUnit).Error

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
			transUnit.ID = dbTransUnit.Identifier

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

func replaceAtIndex(str string, repl string, start int, end int) string {
	strRune := []rune(str)
	replRune := []rune(repl)
	out := []rune{}

	for index := 0; index < start; index++ {
		out = append(out, strRune[index])
	}

	for index := 0; index < len(replRune); index++ {
		out = append(out, replRune[index])
	}

	for index := end; index < len(strRune); index++ {
		out = append(out, strRune[index])
	}

	return string(out)
}
