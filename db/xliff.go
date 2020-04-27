package db

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Job struct {
	gorm.Model
	Active bool
}

type File struct {
	gorm.Model
	JobID uint
	Job   Job
	Path  string
}

type TransUnit struct {
	gorm.Model
	FileID         uint
	File           File
	Resname        string
	Path           string
	Qualifier      string
	State          string
	StateQualifier string
	Source         string
	Target         string
	SourceLanguage string
	TargetLanguage string
}

type Note struct {
	gorm.Model
	TransUnitID uint
	TransUnit   TransUnit
	Data        string
	Language    string
	From        string
}

func OpenDatabase(databaseDialect string, databaseConnection string) (database *gorm.DB, err error) {
	database, err = gorm.Open(databaseDialect, databaseConnection)
	if err != nil {
		return
	}

	database.LogMode(true)

	// Migrate the schema
	database.AutoMigrate(&Job{})
	database.AutoMigrate(&File{})
	database.AutoMigrate(&TransUnit{})
	database.AutoMigrate(&Note{})

	return
}
