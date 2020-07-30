package pipelinerun

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"generators/pkg/manager"
	"generators/pkg/writer"

	"github.com/spf13/cobra"
)

var configFile string

func applyCommand(kubeconfig string) *cobra.Command {
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply generated configuration with pipelinerun to Kubernetes Cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(configFile) == 0 {
				return errors.New("No input file specified")
			}

			client, err := manager.GetKubeClient(kubeconfig)
			if err != nil {
				return fmt.Errorf("fail to create a Kubernetes client: %w", err)
			}

			buf := new(bytes.Buffer)
			if err := writer.WritePipelineRun(configFile, buf); err != nil {
				return fmt.Errorf("fail to get the generated config from %s: %w", configFile, err)
			}
			return manager.CreateResource(context.Background(), client, buf)
		},
	}
	applyCmd.Flags().StringVarP(&configFile, "filename", "f", "", "spec file")
	return applyCmd
}
