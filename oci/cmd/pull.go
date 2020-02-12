package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tektoncd/experimental/oci/action"
)

// pullCmd represents the push command
var pullCmd = &cobra.Command{
	Use:   "pull [ref] kind name",
	Short: "Will fetch the referenced OCI image and write to stdout the Tekton resource of the specified name and kind",
	Long: `Will pull an OCI image from the provided reference which can include a version, sha, and/or tag. Will then
	write to stdout the Tekton resource with the specified name and kind. 
	
	The results are cached so multiple calls can be made without incurring the fetch cost again.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return action.Pull(args[0], args[1], args[2])
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
