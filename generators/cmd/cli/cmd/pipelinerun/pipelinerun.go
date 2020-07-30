package pipelinerun

import (
	"github.com/spf13/cobra"
)

// Command includes the pipelinerun commands
func Command(kubeconfig string) *cobra.Command {
	prCmd := &cobra.Command{
		Use:   "pipelinerun",
		Short: "Manage the generated config with PipelineRun",
	}
	prCmd.AddCommand(
		applyCommand(kubeconfig),
		deleteCommand(kubeconfig),
		showCmd,
		writeCmd,
	)
	return prCmd
}
