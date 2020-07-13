// Package writer provides a method to write the generated task
// to the yaml document.
package writer

import (
	"fmt"
	"generators/pkg/generator"
	"generators/pkg/parser"
	"io"
	"os"

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
	task := generator.GenerateTask(github)
	pipeline, err := generator.GeneratePipeline(github)
	if err != nil {
		return fmt.Errorf("unable to get the pipeline: %w", err)
	}
	trigger := generator.GenerateTrigger(pipeline)

	// pack up all the objects
	pac := []interface{}{task, pipeline, trigger.TriggerBinding, trigger.TriggerTemplate, trigger.EventListener}

	for _, o := range pac {
		if err := wirteYaml(o, writer); err != nil {
			return err
		}
	}
	return nil
}

func wirteYaml(o interface{}, writer io.Writer) error {
	if _, err := writer.Write([]byte("---\n")); err != nil {
		return err
	}
	data, err := yaml.Marshal(o)
	if err != nil {
		return fmt.Errorf("unable to marshal the object: %w", err)
	}
	_, err = writer.Write(data)
	return err
}
