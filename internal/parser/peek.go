package parser

import (
	"os"

	"gopkg.in/yaml.v3"
)

// PeekKind reads the kind field without schema validation.
func PeekKind(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var doc struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return "", err
	}
	return doc.Kind, nil
}
