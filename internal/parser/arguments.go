package parser

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SourceArgument is one argument of a source function.
// Name is the function argument, in the order written in the YAML map.
type SourceArgument struct {
	Name      string         `yaml:"-"`
	Parameter string         `yaml:"parameter,omitempty"`
	Filter    string         `yaml:"filter,omitempty"`
	Default   any            `yaml:"default,omitempty"`
	Map       map[string]any `yaml:"map,omitempty"`
}

// PriorPeriod shifts an aggregation along a breakout.
// Offset -1 is the previous period of Dimension.
type PriorPeriod struct {
	Dimension string `yaml:"dimension"`
	Offset    int    `yaml:"offset"`
}

// VisibleWhen shows a widget only when Parameter equals Equals.
// Metabase has no dashcard visibility control, so the emitter warns.
type VisibleWhen struct {
	Parameter string `yaml:"parameter"`
	Equals    any    `yaml:"equals"`
}

// Arguments keeps the YAML map order. A table function is called in that order.
type Arguments []SourceArgument

func (a *Arguments) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("arguments must be a map")
	}
	out := make(Arguments, 0, len(value.Content)/2)
	for i := 0; i+1 < len(value.Content); i += 2 {
		var arg SourceArgument
		if err := value.Content[i+1].Decode(&arg); err != nil {
			return err
		}
		arg.Name = value.Content[i].Value
		out = append(out, arg)
	}
	*a = out
	return nil
}
