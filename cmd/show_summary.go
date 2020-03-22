package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmwalaszek/katyusha/katyusha"
)

// showSummaryCmd represents the showSummary command
var showSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show summaries associated with given benchmark configuration id",
	Run: func(cmd *cobra.Command, args []string) {
		// workaround for https://github.com/spf13/viper/issues/233
		viper.BindPFlag("id", cmd.Flags().Lookup("id"))

		inv, err := katyusha.NewInventory(viper.GetString("db"))
		if err != nil {
			log.Fatalf("Can't create database file: %v", err)
		}

		summaries, err := inv.FindSummaryForBenchmark(context.Background(), viper.GetInt64("id"))
		if err != nil {
			log.Fatalf("Coould not receive benchmark summary; %v", err)
		}

		fmt.Printf("Found %d summaries for given benchmark\n", len(summaries))

		for i, sm := range summaries {
			fmt.Printf("Summary %d\n", i)
			fmt.Printf("%s\n", sm)
		}
	},
}

func init() {
	showSummaryCmd.Flags().Int64P("id", "i", 0, "Benchmark ID")

	showSummaryCmd.MarkFlagRequired("id")
	viper.BindPFlags(showSummaryCmd.Flags())

	showCmd.AddCommand(showSummaryCmd)
}
