package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tektoncd/experimental/oci/pkg/action"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push [ref] file1 file2 directory/ ...",
	Short: "Will generate an OCI image from the passed in Tekton files or directories",
	Long: `Will create and push a new OCI image from the provided set of file and directory paths. Each file must contain
	a single, parseable Tekton spec such as yaml or json. All directories will be recursively scanned.

	The resulting bundle will be pushed to the referenced image repository using any provided tags. Just like docker, if
	you don't specify a tag, it will be available as :latest. Also, each image will be available with an @sha256 hash.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return action.Push(args[0], args[1:])
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
