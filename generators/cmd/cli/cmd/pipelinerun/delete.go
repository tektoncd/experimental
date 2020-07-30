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

var deleteFile string

func deleteCommand(kubeconfig string) *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete generated configuration with pipelinerun from Kubernetes Cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(deleteFile) == 0 {
				return errors.New("No input file specified")
			}

			client, err := manager.GetKubeClient(kubeconfig)
			if err != nil {
				return fmt.Errorf("fail to create a Kubernetes client: %w", err)
			}

			buf := new(bytes.Buffer)
			if err := writer.WritePipelineRun(deleteFile, buf); err != nil {
				return fmt.Errorf("fail to get the generated config from %s: %w", deleteFile, err)
			}
			return manager.DeleteResource(context.Background(), client, buf)
		},
	}
	deleteCmd.Flags().StringVarP(&deleteFile, "filename", "f", "", "spec file")
	return deleteCmd
}
