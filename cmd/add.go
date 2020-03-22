package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmwalaszek/katyusha/katyusha"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add testcase",
	Long:  "Add testcase. At the moment you can only add testcase from yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		inv, err := katyusha.NewInventory(viper.GetString("db"))
		if err != nil {
			log.Fatalf("Can't create database file: %v", err)
		}

		viper.SetConfigFile(viper.GetString("file"))
		if err := viper.MergeInConfig(); err != nil {
			log.Fatalf("Error on loading benchmark configuration: %v\n", err)
		}

		benchmarkParams, err := benchmarkOptionsToStruct()
		if err != nil {
			log.Fatalf("Benchmark configuration error: %v", err)
		}

		bcID, err := inv.InsertBenchmarkConfiguration(context.Background(), benchmarkParams, viper.GetString("description"))
		if err != nil {
			log.Fatalf("Error inserting benchmark configuration: %v", err)
		}

		log.Printf("Testcase added successfuly with id: %d", bcID)

	},
}

func init() {
	addCmd.Flags().StringP("file", "f", "", "Benchmark file")

	addCmd.MarkFlagRequired("file")
	viper.BindPFlags(addCmd.Flags())

	inventoryCmd.AddCommand(addCmd)
}
