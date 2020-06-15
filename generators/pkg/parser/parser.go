// Package parser provides a method to parse the yaml file
// to the simple products struct.
package parser

import (
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type products struct {
	Items []int    `json:"items,omitempty"`
	Names []string `json:"names,omitempty"`
}

// Parse parses a yaml file from io.Reader and stores the
// result in the products struct
func Parse(r io.Reader) (products, error) {
	reader := yaml.NewYAMLToJSONDecoder(r)
	res := products{}
	err := reader.Decode(&res)
	if err != nil {
		return res, err
	}
	return res, nil
}
