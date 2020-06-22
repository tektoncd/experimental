// Package writer provides a method to write the generated task
// to the yaml document.
package writer

import (
	"fmt"
	"io"
	"os"

	"generators/pkg/generator"
	"generators/pkg/parser"

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

	spec, err := parser.Parse(file)
	if err != nil {
		return fmt.Errorf("unable to parse the file from %s: %w", filename, err)
	}

	task := generator.GenerateTask(spec)
	data, err := yaml.Marshal(&task)
	if err != nil {
		return fmt.Errorf("unable to marshal the file from %s: %w", filename, err)
	}

	_, err = writer.Write(data)
	return err
}
