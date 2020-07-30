// Package writer provides a method to write the generated task
// to the yaml document.
package writer

import (
	"fmt"
	"generators/pkg/generator"
	"generators/pkg/parser"
	"io"
	"os"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func parseFile(filename string) (*generator.GitHub, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open the file from %s. check if the file exists: %w", filename, err)
	}
	defer file.Close()

	github, err := parser.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("unable to parse the file from %s: %w", filename, err)
	}
	return github, nil
}

func getResource(g *generator.GitHub) (*v1beta1.Task, *v1beta1.Pipeline, error) {
	task, err := generator.GenerateTask(g)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get the task: %w", err)
	}
	pipeline, err := generator.GeneratePipeline(g)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get the pipeline: %w", err)
	}
	return task, pipeline, nil
}

// WriteTrigger writes the generated task, pipeline and trigger
// to the io.Writer
func WriteTrigger(filename string, writer io.Writer) error {
	g, err := parseFile(filename)
	if err != nil {
		return fmt.Errorf("unable to parse the github config: %w", err)
	}
	// resrouces needed to write to disk
	t, p, err := getResource(g)
	if err != nil {
		return fmt.Errorf("unable to get the task or pipeline: %w", err)
	}
	trigger, err := generator.GenerateTrigger(p, g)
	if err != nil {
		return fmt.Errorf("unable to get the trigger: %w", err)
	}

	// write into the disk
	return writeYaml(writer, t, p, trigger.TriggerBinding[0], trigger.TriggerBinding[1], trigger.TriggerTemplate, trigger.EventListener)
}

// WriteTrigger writes the generated task, pipeline and pipelinerun
// to the io.Writer
func WritePipelineRun(filename string, writer io.Writer) error {
	g, err := parseFile(filename)
	if err != nil {
		return fmt.Errorf("unable to parse the github config: %w", err)
	}
	// resrouces needed to write to disk
	t, p, err := getResource(g)
	if err != nil {
		return fmt.Errorf("unable to get the task or pipeline: %w", err)
	}
	pr, err := generator.GeneratePipelineRun(p, g)
	if err != nil {
		return fmt.Errorf("unable to get the pipelinerun: %w", err)
	}
	// write into the disk
	return writeYaml(writer, t, p, pr)
}

func writeYaml(writer io.Writer, objs ...runtime.Object) error {
	for _, o := range objs {
		if _, err := writer.Write([]byte("---\n")); err != nil {
			return err
		}
		data, err := yaml.Marshal(o)
		if err != nil {
			return fmt.Errorf("unable to marshal the %s: %w", o.GetObjectKind().GroupVersionKind().Kind, err)
		}
		if _, err = writer.Write(data); err != nil {
			return err
		}
	}
	return nil
}
