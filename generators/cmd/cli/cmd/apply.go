package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"generators/pkg/manager"
	"generators/pkg/writer"

	"github.com/spf13/cobra"
)

var (
	configFile string
	applyCmd   = &cobra.Command{
		Use:   "apply",
		Short: "Apply generated configuration to Kubernetes Cluster",
		RunE:  apply,
	}
)

func init() {
	applyCmd.Flags().StringVarP(&configFile, "filename", "f", "", "spec file")
	rootCmd.AddCommand(applyCmd)
}

func apply(cmd *cobra.Command, args []string) error {
	if len(configFile) == 0 {
		return errors.New("No input file specified")
	}

	client, err := manager.GetKubeClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("fail to create a Kubernetes client: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := writer.WriteToDisk(configFile, buf); err != nil {
		return fmt.Errorf("fail to get the generated config from %s: %w", configFile, err)
	}
	return manager.CreateResource(context.Background(), client, buf)
}
