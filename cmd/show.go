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
func NewShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show benchmark configuration or benchmark confiugration summaries",
		Run: func(cmd *cobra.Command, args []string) {
			// workaround for https://github.com/spf13/viper/issues/233
			viper.BindPFlag("id", cmd.Flags().Lookup("id"))
			idChanged := cmd.Flags().Lookup("id").Changed

			var bcs []*katyusha.BenchmarkConfiguration
			var err error

			inv, err := katyusha.NewInventory(viper.GetString("db"))
			if err != nil {
				log.Fatalf("Can't create database file: %v", err)
			}

			if id := viper.GetInt64("id"); id != 0 {
				bc, err := inv.FindBenchmarkByID(context.Background(), id)
				if err != nil {
					log.Fatalf("Can't get benchamrks from the database: %v", err)
				}

				bcs = append(bcs, bc)
			} else if url := viper.GetString("url"); url != "" {
				bcs, err = inv.FindBenchmarkByURL(context.Background(), url)
				if err != nil {
					log.Fatalf("Can't get benchmark from the database: %v", err)
				}
			} else if viper.GetBool("all") && !idChanged {
				bcs, err = inv.FindAllBenchmarks(context.Background())
				if err != nil {
					log.Fatalf("Can't get benchamrks from the database: %v", err)
				}
			}

			fmt.Printf("Found %d benchmarks\n", len(bcs))

			for i, bc := range bcs {
				fmt.Printf("Benchmark [%d]\n", i+1)
				fmt.Println(bc)
				fmt.Printf("\n")

				if viper.GetBool("full") {
					summaries, err := inv.FindSummaryForBenchmark(context.Background(), bc.ID)
					if err != nil {
						log.Fatalf("Can't get benchmark summary: %v", err)
					}

					fmt.Println("Summaries: ")
					for idx, summary := range summaries {
						fmt.Printf("[%d] \n", idx+1)
						fmt.Println(summary)
						fmt.Printf("\n")
					}
				}
			}
		},
	}

	cmd.Flags().Int64P("id", "i", 0, "Benchmark confiugration id")
	cmd.Flags().StringP("url", "u", "", "Benchmark URL")
	cmd.Flags().BoolP("all", "a", true, "Show all benchmarks")
	cmd.Flags().BoolP("full", "f", false, "Show benchmark configuration with test results")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func init() {
	cmd := NewShowCmd()
	inventoryCmd.AddCommand(cmd)
}
