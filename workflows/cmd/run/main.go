package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/tektoncd/experimental/workflows/pkg/client/clientset/versioned/scheme"
	"github.com/tektoncd/experimental/workflows/pkg/convert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"

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

func runWorkflow(fileName string) error {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("error reading file: %+v", err)
	}

	sch := runtime.NewScheme()
	_ = scheme.AddToScheme(sch)

	decoder := streaming.NewDecoder(ioutil.NopCloser(bytes.NewReader(file)), serializer.NewCodecFactory(sch).UniversalDecoder())
	w := new(v1alpha1.Workflow)
	_, _, err = decoder.Decode(nil, w)
	if err != nil {
		if err != io.EOF {
			return fmt.Errorf("error decoding workflow: %v", err)
		}
	}
	tt, err := convert.ToPipelineRun(w)
	if err != nil {
		return fmt.Errorf("error converting to pipelineRun: %s", err)
	}
	tty, err := yaml.Marshal(tt)
	if err != nil {
		return fmt.Errorf("error convering pipelineRun to yaml: %w", err)
	}
	fmt.Printf("%s", tty)
	return nil
}
