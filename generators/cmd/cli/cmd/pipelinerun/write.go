package pipelinerun

import (
	"errors"
	"generators/pkg/writer"
	"os"

	"github.com/spf13/cobra"
)

var (
	specFilename   string
	outputFilename string
	writeCmd       = &cobra.Command{
		Use:   "write",
		Short: "Write generated configuration with pipelinrun to disk.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(specFilename) == 0 {
				return errors.New("No input file specified")
			}

			file, err := os.Create(outputFilename)
			if err != nil {
				return err
			}
			defer file.Close()

			return writer.WritePipelineRun(specFilename, file)
		},
	}
)

func init() {
	writeCmd.Flags().StringVarP(&specFilename, "filename", "f", "", "input spec file")
	writeCmd.Flags().StringVarP(&outputFilename, "output", "o", "gen-config.yaml", "generated config file")
}
