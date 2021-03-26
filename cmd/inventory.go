package cmd

import (
	"github.com/spf13/cobra"
)

// inventoryCmd represents the inventory command
var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Testcases and test summary inventory management",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}
