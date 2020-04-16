package commands

import (
	"github.com/dragosv/delta/xliff"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"os"
)

type TransUnit struct {
	gorm.Model
	Code  string
	Price uint
}

type XliffTransUnit struct {
	Path string
	Unit xliff.TransUnit
}

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
var transUnits []XliffTransUnit

func init() {
	rootCmd.AddCommand(pushCommand)
}

func runPushCommand() {
	jww.FEEDBACK.Println("push")

	db, err := gorm.Open(databaseDialect, databaseConnection)
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&TransUnit{})

	fs = afero.NewOsFs()
	transUnits := []XliffTransUnit{}

	afero.Walk(fs, source, pushWalkFunc)

	for _, transUnit := range transUnits {
		jww.FEEDBACK.Println(transUnit.Unit.Source)
	}
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
		var incompleteTransUnits = document.IncompleteTransUnits()
		for _, xliffTransUnit := range incompleteTransUnits {
			var transUnit = XliffTransUnit{Path: path, Unit: xliffTransUnit}

			transUnits = append(transUnits, transUnit)
		}
	}

	return nil
}
