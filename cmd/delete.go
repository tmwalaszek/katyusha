package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deleteCmd represents the delete command
func NewDeleteCmd(inv Inventory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete benchmark configurations with all data associated",
		Run: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("id", cmd.Flags().Lookup("id"))

			err := inv.DeleteBenchmark(context.Background(), viper.GetInt64("id"))
			if err != nil {
				log.Fatalf("Can't remove benchmark configuration: %v", err)
			}

			log.Printf("Benchmark with all data removed\n")
		},
	}

	cmd.Flags().Int64P("id", "i", 0, "Benchmark configuration ID")
	cmd.MarkFlagRequired("id")
	viper.BindPFlags(cmd.Flags())

	return cmd
}
