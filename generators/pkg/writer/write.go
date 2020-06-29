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

	task := generator.GenerateTask(github)
	pipeline, err := generator.GeneratePipeline(github)

	// write task
	if _, err := writer.Write([]byte("---\n")); err != nil {
		return err
	}
	data, err := yaml.Marshal(&task)
	if err != nil {
		return fmt.Errorf("unable to marshal the task %s: %w", task.Name, err)
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}

	// write pipeline
	if _, err := writer.Write([]byte("---\n")); err != nil {
		return err
	}
	data, err = yaml.Marshal(&pipeline)
	if err != nil {
		return fmt.Errorf("unable to marshal the pipeline %s: %w", pipeline.Name, err)
	}
	_, err = writer.Write(data)
	return err
}
