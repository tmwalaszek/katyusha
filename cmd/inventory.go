package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmwalaszek/katyusha/katyusha"
)

// inventoryCmd represents the inventory command
var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Testcases and test summary inventory management",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	rootCmd.AddCommand(inventoryCmd)

	inv, err := katyusha.NewInventory(viper.GetString("db"))
	if err != nil {
		log.Fatalf("Can't create database file: %v", err)
	}

	showCmd := NewShowCmd(inv)
	addCmd := NewAddCmd(inv)
	runCmd := NewRunCmd(inv)
	deleteCmd := NewDeleteCmd(inv)

	inventoryCmd.AddCommand(showCmd, addCmd, runCmd, deleteCmd)
}
