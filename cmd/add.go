package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addCmd represents the add command
func NewAddCmd(inv Inventory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add testcase",
		Long:  "Add testcase. At the moment you can only add testcase from yaml file",
		Run: func(cmd *cobra.Command, args []string) {
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

	cmd.Flags().StringP("file", "f", "", "Benchmark file")

	cmd.MarkFlagRequired("file")
	viper.BindPFlags(cmd.Flags())

	return cmd
}
