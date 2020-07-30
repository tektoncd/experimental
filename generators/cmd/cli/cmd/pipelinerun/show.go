package pipelinerun

import (
	"errors"
	"generators/pkg/writer"
	"os"

	"github.com/spf13/cobra"
)

var (
	filename string
	showCmd  = &cobra.Command{
		Use:   "show",
		Short: "Show generated configuration with pipelinerun.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(filename) == 0 {
				return errors.New("No input file specified")
			}

			if err := writer.WritePipelineRun(filename, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	showCmd.Flags().StringVarP(&filename, "filename", "f", "", "spec file")
}
