package commands

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/dragosv/delta/db"
	"github.com/dragosv/delta/job"
	"github.com/dragosv/delta/xliff"
	"github.com/jinzhu/gorm"
	"github.com/spf13/afero"
	"os"
	p "plugin"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile            string
	source             string
	databaseDialect    string
	databaseConnection string
	destination        string
	plugin             string
	config             string

	rootCmd = &cobra.Command{
		Use:   "delta",
		Short: "Delta Command Line Interface",
		Long:  `Delta is a plugable Command Line Interface for job based Translation Management Services.`,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.delta)")

	rootCmd.Flags().StringVarP(&source, "source", "s", "", "Source directory to read from")
	rootCmd.Flags().StringVarP(&destination, "destination", "d", "", "Destination directory to write to")
	rootCmd.Flags().StringVarP(&databaseDialect, "dialect", "", "", "Database dialect")
	rootCmd.Flags().StringVarP(&databaseConnection, "connection", "", "", "Database connection string")
	rootCmd.Flags().StringVarP(&plugin, "plugin", "p", "", "Job plugin")
	rootCmd.Flags().StringVarP(&config, "config", "c", "", "Job plugin configuration file")

	viper.BindPFlag("source", rootCmd.PersistentFlags().Lookup("source"))
	viper.BindPFlag("destination", rootCmd.PersistentFlags().Lookup("destination"))
	viper.BindPFlag("dialect", rootCmd.PersistentFlags().Lookup("dialect"))
	viper.BindPFlag("connection", rootCmd.PersistentFlags().Lookup("connection"))
	viper.BindPFlag("plugin", rootCmd.PersistentFlags().Lookup("plugin"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			er(err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".delta")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func openDatabase(databaseDialect string, databaseConnection string) (database *gorm.DB, err error) {
	database, err = db.OpenDatabase(databaseDialect, databaseConnection)

	return
}

func getJob() (job.Job, error) {
	pluginObject, error := p.Open(plugin)

	if error != nil {
		return nil, error
	}

	symJob, symJobError := pluginObject.Lookup("Job")

	if symJobError != nil {
		return nil, error
	}

	job, ok := symJob.(job.Job)

	if !ok {
		return nil, errors.New("unexpected type from module symbol")
	}

	return job, nil
}

func writeDocument(document xliff.Document, path string) error {
	file, err := xml.MarshalIndent(document, "", " ")

	if err != nil {
		var language string

		if len(document.Files) > 0 {
			language = document.Files[0].TargetLanguage
		}

		return errors.New("failed to write xliff document for language " + language)
	}

	err = afero.WriteFile(fs, path, file, 0644)

	if err != nil {
		return errors.New("failed to write xliff file " + path)
	}

	return nil
}
