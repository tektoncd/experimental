package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tkn-oci",
	Short: "Pushes and fetches Tekton resources from OCI images.",
	Long: `tkn-oci will allow you to fetch and push Tekton pipelines, task, etc
	from an OCI-compliant image repository like docker. This allows you to specify
	a specific version of that task or pipeline spec using a tag or sha hash.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
