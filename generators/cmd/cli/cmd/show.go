package cmd

import (
	"generators/pkg/writer"
	"os"

	"github.com/spf13/cobra"
)

var filename string
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show generated configuration.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(filename) == 0 {
			cmd.Help()
			return nil
		}

		if err := writer.WriteToDisk(filename, os.Stdout); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	showCmd.Flags().StringVarP(&filename, "filename", "f", "", "spec file")
	rootCmd.AddCommand(showCmd)
}
