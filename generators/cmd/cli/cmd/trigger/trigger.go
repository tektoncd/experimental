package trigger

import (
	"github.com/spf13/cobra"
)

// Command includes the trigger commands
func Command(kubeconfig string) *cobra.Command {
	trCmd := &cobra.Command{
		Use:   "trigger",
		Short: "Manage the generated config with Trigger",
	}
	trCmd.AddCommand(
		applyCommand(kubeconfig),
		deleteCommand(kubeconfig),
		showCmd,
		writeCmd,
	)
	return trCmd
}
