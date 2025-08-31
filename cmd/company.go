package cmd

import (
	"github.com/spf13/cobra"
)

// companyCmd represents the company command
var companyCmd = &cobra.Command{
	Use:   "company",
	Short: "Manage companies",
}

func init() {
	rootCmd.AddCommand(companyCmd)
}
