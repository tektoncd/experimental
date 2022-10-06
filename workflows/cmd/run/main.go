package main

import (
	"fmt"
	"io/ioutil"

	"github.com/tektoncd/experimental/workflows/pkg/client/clientset/versioned/scheme"
	"github.com/tektoncd/experimental/workflows/pkg/convert"
	"sigs.k8s.io/yaml"

	"github.com/spf13/cobra"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
)

func main() {
	var fileName string

	var runCmd = &cobra.Command{
		Use: "run a workflow from a file",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runWorkflow(fileName); err != nil {
				fmt.Println(err.Error())
			}
		},
	}

	runCmd.Flags().StringVarP(&fileName, "file", "f", "", "workflow.yaml to use")
	runCmd.MarkFlagRequired("file")
	var rootCmd = &cobra.Command{
		Use:  "workflow",
		Args: cobra.MinimumNArgs(1),
	}
	rootCmd.AddCommand(runCmd)
	rootCmd.Execute()
}

func parseWorkflowOrDie(yaml []byte) *v1alpha1.Workflow {
	var w v1alpha1.Workflow
	meta := `apiVersion: tekton.dev/v1alpha1
kind: Workflow
`
	bytes := append([]byte(meta), yaml...)
	if _, _, err := scheme.Codecs.UniversalDeserializer().Decode(bytes, nil, &w); err != nil {
		panic(fmt.Sprintf("failed to parse workflow: %s", err))
	}
	return &w
}

func runWorkflow(fileName string) error {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("error reading file: %+v", err)
	}
	w := parseWorkflowOrDie(file)
	tt, err := convert.ToPipelineRun(w)
	if err != nil {
		return fmt.Errorf("error converting to pipelineRun: %s", err)
	}
	tty, err := yaml.Marshal(tt)
	if err != nil {
		return fmt.Errorf("error converting pipelineRun to yaml: %w", err)
	}
	fmt.Printf("%s", tty)
	return nil
}
