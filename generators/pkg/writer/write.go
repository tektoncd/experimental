// Package writer provides a method to write the generated task
// to the yaml document.
package writer

import (
	"fmt"
	"generators/pkg/generator"
	"generators/pkg/parser"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// WriteToDisk writes the generated task
// to the io.Writer
func WriteToDisk(filename string, writer io.Writer) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("unable to open the file from %s. check if the file exists: %w", filename, err)
	}
	defer file.Close()

	github, err := parser.Parse(file)
	if err != nil {
		return fmt.Errorf("unable to parse the file from %s: %w", filename, err)
	}

	// resrouces needed to write to disk
	task, err := generator.GenerateTask(github)
	if err != nil {
		return fmt.Errorf("unable to get the task: %w", err)
	}

	pipeline, err := generator.GeneratePipeline(github)
	if err != nil {
		return fmt.Errorf("unable to get the pipeline: %w", err)
	}

	trigger, err := generator.GenerateTrigger(pipeline, github)
	if err != nil {
		return fmt.Errorf("unable to get the trigger: %w", err)
	}

	// write into the disk
	return writeYaml(writer, task, pipeline, trigger.TriggerBinding[0], trigger.TriggerBinding[1], trigger.TriggerTemplate, trigger.EventListener)
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
