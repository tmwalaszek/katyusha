package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmwalaszek/katyusha/katyusha"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show benchmark configuration or benchmark confiugration summaries",
	Run: func(cmd *cobra.Command, args []string) {
		// workaround for https://github.com/spf13/viper/issues/233
		viper.BindPFlag("id", cmd.Flags().Lookup("id"))

		var bcs []*katyusha.BenchmarkConfiguration
		var err error

		inv, err := katyusha.NewInventory(viper.GetString("db"))
		if err != nil {
			log.Fatalf("Can't create database file: %v", err)
		}

		if id := viper.GetInt64("id"); id != 0 {
			bcs, err = inv.FindBenchmarkByID(context.Background(), id)
			if err != nil {
				log.Fatalf("Can't get benchamrks from the database: %v", err)
			}
		} else if url := viper.GetString("url"); url != "" {
			bcs, err = inv.FindBenchmarkByURL(context.Background(), url)
			if err != nil {
				log.Fatalf("Can't get benchmark from the database: %v", err)
			}
		} else if viper.GetBool("all") {
			bcs, err = inv.FindAllBenchmarks(context.Background())
			if err != nil {
				log.Fatalf("Can't get benchamrks from the database: %v", err)
			}
		}

		fmt.Printf("Found %d benchmarks\n", len(bcs))

		for i, bc := range bcs {
			fmt.Printf("Benchmark [%d]\n", i+1)
			if viper.GetBool("full") {
				fmt.Println(bc)
			} else {
				fmt.Printf("ID:\t\t %d\nDescription:\t %s\nUrl:\t\t %s\n", bc.ID, bc.Description, bc.URL)
			}
			fmt.Printf("\n")
		}
	},
}

func init() {
	showCmd.Flags().Int64P("id", "i", 0, "Benchmark confiugration id")
	showCmd.Flags().StringP("url", "u", "", "Benchmark URL")
	showCmd.Flags().BoolP("all", "a", true, "Show all benchmarks")
	showCmd.Flags().BoolP("full", "f", false, "Show benchmark configuration with test results")

	viper.BindPFlags(showCmd.Flags())

	inventoryCmd.AddCommand(showCmd)
}
