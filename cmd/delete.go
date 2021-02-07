package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmwalaszek/katyusha/katyusha"
)

// deleteCmd represents the delete command
func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete benchmark configurations with all data associated",
		Run: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("id", cmd.Flags().Lookup("id"))

			inv, err := katyusha.NewInventory(viper.GetString("db"))
			if err != nil {
				log.Fatalf("Can't initialize database: %v", err)
			}

			err = inv.DeleteBenchmark(context.Background(), viper.GetInt64("id"))
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

func init() {
	cmd := NewDeleteCmd()
	inventoryCmd.AddCommand(cmd)
}
