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
		case "number":
			dim["type"] = "number"
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
		}
		if meas.Expression != "" {
			ex, err := parser.ParseExpression(meas.Expression)
			if err != nil {
				return nil, fmt.Errorf("measure %s: %w", meas.Name, err)
			}
			mm["type"] = "number"
			mm["sql"] = parser.FormatExpr(ex, func(name string) string { return "{" + name + "}" })
		} else if meas.Aggregation == "ltm" {
			mm["sql"] = meas.Field
			mm["type"] = "sum"
			mm["rolling_window"] = map[string]any{
				"trailing": "12 month",
				"offset":   "end",
			}
		} else {
			mm["sql"] = meas.Field
			mm["type"] = mapAgg(meas.Aggregation)
		}
		if len(meas.Where) > 0 {
			filters := make([]map[string]any, 0, len(meas.Where))
			for _, p := range meas.Where {
				filters = append(filters, map[string]any{"sql": predicateSQL(p)})
			}
			mm["filters"] = filters
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

func predicateSQL(p parser.Predicate) string {
	col := "{CUBE}." + p.Dimension
	switch p.Operator {
	case "neq":
		return col + " != " + sqlLiteral(p.Value)
	case "gt":
		return col + " > " + sqlLiteral(p.Value)
	case "gte":
		return col + " >= " + sqlLiteral(p.Value)
	case "lt":
		return col + " < " + sqlLiteral(p.Value)
	case "lte":
		return col + " <= " + sqlLiteral(p.Value)
	case "in":
		return col + " IN (" + sqlList(p.Value) + ")"
	case "not_in":
		return col + " NOT IN (" + sqlList(p.Value) + ")"
	default:
		return col + " = " + sqlLiteral(p.Value)
	}
}

func sqlList(v any) string {
	items, ok := v.([]any)
	if !ok {
		return sqlLiteral(v)
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, sqlLiteral(item))
	}
	return strings.Join(parts, ", ")
}

func sqlLiteral(v any) string {
	switch t := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(t, "'", "''") + "'"
	case int, int64, float64, bool:
		return fmt.Sprint(t)
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(t), "'", "''") + "'"
	}
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
