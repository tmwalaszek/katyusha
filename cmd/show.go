package cmd

import (
	"github.com/spf13/cobra"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show benchmark configuration or benchmark confiugration summaries",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	inventoryCmd.AddCommand(showCmd)
}
