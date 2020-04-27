package commands

import (
	"fmt"
	"github.com/dragosv/delta/db"
	"github.com/jinzhu/gorm"
	"os"

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
	rootCmd.Flags().StringVarP(&destination, "destionation", "d", "", "Destination directory to write to")
	rootCmd.Flags().StringVarP(&databaseDialect, "dialect", "", "", "Database dialect")
	rootCmd.Flags().StringVarP(&databaseConnection, "connection", "", "", "Database connection string")

	viper.BindPFlag("source", rootCmd.PersistentFlags().Lookup("source"))
	viper.BindPFlag("destination", rootCmd.PersistentFlags().Lookup("destination"))
	viper.BindPFlag("dialect", rootCmd.PersistentFlags().Lookup("dialect"))
	viper.BindPFlag("connection", rootCmd.PersistentFlags().Lookup("connection"))
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
