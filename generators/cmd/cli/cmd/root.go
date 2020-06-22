package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tkn-gen",
	Short: "This is the CLI for tekton generators",
	Long: `tkn-gen will allow you to generate Tekton spec from the given simple spec. 
			It also contains support for managing generated configuration.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
