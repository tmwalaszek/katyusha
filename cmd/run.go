package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmwalaszek/katyusha/katyusha"
)

// deleteCmd represents the delete command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run benchmark from inventory",
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("id", cmd.Flags().Lookup("id"))
		viper.BindPFlag("save", cmd.Flags().Lookup("save"))

		inv, err := katyusha.NewInventory(viper.GetString("db"))
		if err != nil {
			log.Fatalf("Can't initialize database: %v", err)
		}

		bc, err := inv.FindBenchmarkByID(context.Background(), viper.GetInt64("id"))
		if err != nil {
			log.Fatalf("Can't get benchmark from inventory: %v", err)
		}

		if bc == nil {
			log.Fatalf("Benchmark %d does not exists", viper.GetInt64("id"))
		}

		runBenchmark(&bc.BenchmarkParameters)
	},
}

func init() {
	runCmd.Flags().Int64P("id", "i", 0, "Benchmark configuration ID")
	runCmd.Flags().BoolP("save", "s", false, "Save benchmark configuration and result")
	runCmd.Flags().StringP("description", "d", "", "Test run description")

	runCmd.MarkFlagRequired("id")

	viper.BindPFlags(runCmd.Flags())
	inventoryCmd.AddCommand(runCmd)
}
