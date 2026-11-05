package cube

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

// Emit produces a Cube-compatible YAML data model from a SemanticModel.
// Output is a single YAML document under a "cubes:" key.
func Emit(m *parser.SemanticModel) ([]byte, error) {
	cube := map[string]any{
		"name": m.Metadata.Name,
	}
	if m.Spec.Source != nil {
		if m.Spec.Source.Table != "" {
			cube["sql_table"] = qualify(m.Spec.Source)
		}
	}
	if m.Metadata.Description != "" {
		cube["description"] = m.Metadata.Description
	}

	// Dimensions
	var dims []map[string]any
	for _, d := range m.Spec.Dimensions {
		dim := map[string]any{
			"name": d.Name,
			"sql":  d.Field,
		}
		switch d.Type {
		case "time":
			dim["type"] = "time"
		case "categorical":
			dim["type"] = "string"
		default:
			dim["type"] = "string"
		}
		if d.Title != "" {
			dim["title"] = d.Title
		}
		dims = append(dims, dim)
	}
	if len(dims) > 0 {
		cube["dimensions"] = dims
	}

	// Measures
	var measures []map[string]any
	for _, meas := range m.Spec.Measures {
		mm := map[string]any{
			"name": meas.Name,
			"sql":  meas.Field,
			"type": mapAgg(meas.Aggregation),
		}
		if meas.Title != "" {
			mm["title"] = meas.Title
		}
		if meas.Description != "" {
			mm["description"] = meas.Description
		}
		measures = append(measures, mm)
	}
	// Metrics that are ratios become calculated measures
	for _, met := range m.Spec.Metrics {
		if met.Type == "ratio" && met.Numerator != "" && met.Denominator != "" {
			measures = append(measures, map[string]any{
				"name":  met.Name,
				"type":  "number",
				"sql":   fmt.Sprintf("{%s} / {%s}", met.Numerator, met.Denominator),
				"title": met.Title,
			})
		}
	}
	if len(measures) > 0 {
		cube["measures"] = measures
	}

	out := map[string]any{
		"cubes": []any{cube},
	}
	return yaml.Marshal(out)
}

func qualify(s *parser.Source) string {
	parts := []string{}
	if s.Database != "" {
		parts = append(parts, s.Database)
	}
	if s.Schema != "" {
		parts = append(parts, s.Schema)
	}
	if s.Table != "" {
		parts = append(parts, s.Table)
	}
	return strings.Join(parts, ".")
}

func mapAgg(a string) string {
	switch a {
	case "count_distinct":
		return "count_distinct"
	case "count":
		return "count"
	case "avg":
		return "avg"
	case "min":
		return "min"
	case "max":
		return "max"
	default:
		return "sum"
	}
}
