package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRunCmd(benchmark Benchmark, inv Inventory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run benchmark from inventory",
		Run: func(cmd *cobra.Command, args []string) {
			logger := log.New(cmd.ErrOrStderr(), "", log.LstdFlags)
			viper.BindPFlag("id", cmd.Flags().Lookup("id"))
			viper.BindPFlag("save", cmd.Flags().Lookup("save"))

			bc, err := inv.FindBenchmarkByID(context.Background(), viper.GetInt64("id"))
			if err != nil {
				log.Fatalf("Can't get benchmark from inventory: %v", err)
			}

			if bc == nil {
				log.Fatalf("Benchmark %d does not exists", viper.GetInt64("id"))
			}

			runBenchmark(benchmark, inv, &bc.BenchmarkParameters, logger)
		},
	}

	cmd.Flags().Int64P("id", "i", 0, "Benchmark configuration ID")
	cmd.Flags().BoolP("save", "s", false, "Save benchmark configuration and result")
	cmd.Flags().StringP("description", "d", "", "Test run description")

	cmd.MarkFlagRequired("id")

	viper.BindPFlags(cmd.Flags())

	return cmd
}
