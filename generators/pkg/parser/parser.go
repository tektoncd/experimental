// Package parser provides a method to parse the yaml file
// to the simple products struct.
package parser

import (
	"fmt"
	"generators/pkg/generator"
	"io"
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

// Parse parses a yaml file from io.Reader and stores the
// result in the GitHub struct
func Parse(r io.Reader) (*generator.GitHub, error) {
	github, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, fmt.Errorf("fail to read from the input: %w", err)
	}
	res := generator.GitHub{}
	if err := yaml.Unmarshal(github, &res); err != nil {
		return &res, fmt.Errorf("fail to unmarshal from the input: %w", err)
	}

	return &res, nil
}
