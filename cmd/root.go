package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmwalaszek/katyusha/katyusha"
)

type Benchmark interface {
	StartBenchmark(ctx context.Context) *katyusha.Summary
	ConfigureBenchmark(*katyusha.BenchmarkParameters) error
}

type Inventory interface {
	Close() error
	FindAllBenchmarks(ctx context.Context) ([]*katyusha.BenchmarkConfiguration, error)
	FindBenchmarkByID(ctx context.Context, ID int64) (*katyusha.BenchmarkConfiguration, error)
	FindBenchmark(ctx context.Context, URL string, description string) (*katyusha.BenchmarkConfiguration, error)
	FindBenchmarkByURL(ctx context.Context, URL string) ([]*katyusha.BenchmarkConfiguration, error)
	FindSummaryForBenchmark(ctx context.Context, bcID int64) ([]*katyusha.BenchmarkSummary, error)
	DeleteBenchmark(ctx context.Context, bcID int64) error
	InsertBenchmarkSummary(ctx context.Context, summary *katyusha.Summary, description string, bcId int64) error
	InsertBenchmarkConfiguration(ctx context.Context, benchParameters *katyusha.BenchmarkParameters, description string) (int64, error)
}

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

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	defaultConfFile := path.Join(home, ".katyusha", "katyusha.yaml")
	defaultDbFile := path.Join(home, ".katyusha", "inventory.db")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfFile, "config file")
	rootCmd.PersistentFlags().StringVar(&dbFile, "db", defaultDbFile, "Inventory file location")

	viper.BindPFlag("db", rootCmd.PersistentFlags().Lookup("db"))

	cobra.OnInitialize(initConfig)

	benchmark := katyusha.NewBenchmark()
	inv, err := katyusha.NewInventory(viper.GetString("db"))
	if err != nil {
		log.Fatalf("Can't create database file: %v", err)
	}

	showCmd := NewShowCmd(inv)
	addCmd := NewAddCmd(inv)
	runCmd := NewRunCmd(benchmark, inv)
	deleteCmd := NewDeleteCmd(inv)
	benchmarkCmd := NewBenchmarkCmd(benchmark, inv)

	inventoryCmd.AddCommand(showCmd, addCmd, runCmd, deleteCmd)
	rootCmd.AddCommand(inventoryCmd, benchmarkCmd)
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
