package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	dbFile  string
)

const (
	Method      string = "GET"
	Requests    int    = 1000
	Connections int    = 10
	Abort       int    = 1000
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "example",
	Short: "Katyusha CLI",
	Long: `Katyusha CLI is a HTTP benchmarking tool written in Golang.
It uses fasthttp library to make HTTP requests.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	viper.SetDefault("method", Method)
	viper.SetDefault("requests", Requests)
	viper.SetDefault("connections", Connections)
	viper.SetDefault("abort", Abort)

	cobra.OnInitialize(initConfig)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	defaultConfFile := path.Join(home, ".katyusha", "katyusha.yaml")
	defaultDbFile := path.Join(home, ".katyusha", "inventory.db")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfFile, "config file")
	rootCmd.PersistentFlags().StringVar(&dbFile, "db", defaultDbFile, "Inventory file location")

	viper.BindPFlag("db", rootCmd.PersistentFlags().Lookup("db"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix("KATYUSHA")
	viper.AutomaticEnv() // read in environment variables that match

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".example" (without extension).
		if viper.GetString("CONFIG") != "" {
			viper.SetConfigFile(viper.GetString("CONFIG"))
		}
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			return
		}

		log.Fatalf("Could not load katyusha config file: %v", err)
	}
}
