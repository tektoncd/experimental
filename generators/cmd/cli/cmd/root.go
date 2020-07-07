package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

var (
	kubeconfig string
	rootCmd    = &cobra.Command{
		Use:   "tkn-gen",
		Short: "This is the CLI for tekton generators",
		Long: `tkn-gen will allow you to generate Tekton spec from the given simple spec. 
			It also contains support for managing generated configuration.`,
	}
)

func init() {
	if home := homedir.HomeDir(); home != "" {
		rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "absolute path to the kubeconfig file")
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
