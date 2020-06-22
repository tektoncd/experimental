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
// result in the GitHubSpec struct
func Parse(r io.Reader) (generator.GitHubSpec, error) {
	spec, err := ioutil.ReadAll(r)

	if err != nil {
		return generator.GitHubSpec{}, fmt.Errorf("fail to read from the input: %w", err)
	}
	res := generator.GitHubSpec{}
	if err := yaml.Unmarshal(spec, &res); err != nil {
		return res, fmt.Errorf("fail to unmarshal from the input: %w", err)
	}

	return res, nil
}
