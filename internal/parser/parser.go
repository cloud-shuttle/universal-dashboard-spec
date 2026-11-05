package parser

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Dashboard struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
}

type Spec struct {
	Measures   []Measure   `yaml:"measures"`
	Dimensions []Dimension `yaml:"dimensions"`
	Filters    []Filter    `yaml:"filters,omitempty"`
	Tabs       []Tab       `yaml:"tabs,omitempty"`
	Widgets    []Widget    `yaml:"widgets"`
}

type Measure struct {
	Name        string `yaml:"name"`
	Title       string `yaml:"title,omitempty"`
	Aggregation string `yaml:"aggregation"`
	Field       string `yaml:"field,omitempty"`
	Format      string `yaml:"format,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Dimension struct {
	Name          string `yaml:"name"`
	Title         string `yaml:"title,omitempty"`
	Field         string `yaml:"field,omitempty"`
	Type          string `yaml:"type,omitempty"`
	TemporalGrain string `yaml:"temporalGrain,omitempty"`
}

type Filter struct {
	Name      string `yaml:"name"`
	Title     string `yaml:"title,omitempty"`
	Dimension string `yaml:"dimension"`
	Required  bool   `yaml:"required,omitempty"`
}

type Tab struct {
	Name  string `yaml:"name"`
	Title string `yaml:"title"`
}

type Widget struct {
	ID         string         `yaml:"id"`
	Type       string         `yaml:"type"`
	Title      string         `yaml:"title"`
	Tab        string         `yaml:"tab,omitempty"`
	Measures   []string       `yaml:"measures,omitempty"`
	Dimensions []string       `yaml:"dimensions,omitempty"`
	Filters    []string       `yaml:"filters,omitempty"`
	Layout     Layout         `yaml:"layout"`
	Options    map[string]any `yaml:"options,omitempty"`
}

type Layout struct {
	X int `yaml:"x"`
	Y int `yaml:"y"`
	W int `yaml:"w"`
	H int `yaml:"h"`
}

func Load(path string) (*Dashboard, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d Dashboard
	if err := yaml.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	return &d, nil
}
