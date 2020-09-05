package cmd

import (
	"context"
	"github.com/tmwalaszek/katyusha/katyusha/inventory"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete benchmark configurations with all data associated",
	Run: func(cmd *cobra.Command, args []string) {
		inv, err := inventory.NewInventory("sqlite", viper.GetString("db"))
		if err != nil {
			log.Fatalf("Can't initialize database: %v", err)
		}

		err = inv.DeleteBenchmark(context.Background(), viper.GetInt64("benchmark"))
		if err != nil {
			log.Fatalf("Can't remove benchmark configuration: %v", err)
		}

		log.Printf("Benchmark with all data removed\n")
	},
}

func init() {
	deleteCmd.Flags().Int64P("benchmark", "b", 0, "Benchmark configuration ID")

	deleteCmd.MarkFlagRequired("benchmark")

	viper.BindPFlags(deleteCmd.Flags())
	inventoryCmd.AddCommand(deleteCmd)
}
