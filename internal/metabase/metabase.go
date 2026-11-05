package metabase

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func Emit(d *parser.Dashboard) (map[string][]byte, error) {
	files := map[string][]byte{}

	// Minimal Metabase Representations-style dashboard YAML
	dash := map[string]any{
		"name":        d.Metadata.Title,
		"description": d.Metadata.Description,
		"parameters":  buildParameters(d),
		"dashcards":   buildDashcards(d),
	}

	b, err := yaml.Marshal(dash)
	if err != nil {
		return nil, err
	}
	files[d.Metadata.Name+".dashboard.yaml"] = b

	// One card file per widget (stub — real emitter would emit MBQL)
	for _, w := range d.Spec.Widgets {
		card := map[string]any{
			"name":                  w.Title,
			"display":               mapDisplay(w.Type),
			"visualization_settings": map[string]any{},
			// In a full emitter this would contain the MBQL query
			// built from measures + dimensions.
		}
		if goal, ok := w.Options["goalLine"]; ok {
			card["visualization_settings"] = map[string]any{
				"graph.goal_value": goal,
			}
		}
		cb, err := yaml.Marshal(card)
		if err != nil {
			return nil, err
		}
		files[fmt.Sprintf("card-%s.yaml", w.ID)] = cb
	}

	return files, nil
}

func buildParameters(d *parser.Dashboard) []map[string]any {
	var params []map[string]any
	for _, f := range d.Spec.Filters {
		params = append(params, map[string]any{
			"name":   f.Title,
			"slug":   f.Name,
			"type":   "string/=",
			"target": []any{"dimension", []any{"template-tag", f.Dimension}},
		})
	}
	return params
}

func buildDashcards(d *parser.Dashboard) []map[string]any {
	var cards []map[string]any
	for _, w := range d.Spec.Widgets {
		cards = append(cards, map[string]any{
			"card_id": w.ID, // placeholder — real emitter uses stable NanoID
			"row":     w.Layout.Y,
			"col":     w.Layout.X,
			"size_x":  w.Layout.W,
			"size_y":  w.Layout.H,
		})
	}
	return cards
}

func mapDisplay(t string) string {
	switch t {
	case "line":
		return "line"
	case "bar", "stacked_bar":
		return "bar"
	case "pie":
		return "pie"
	case "table":
		return "table"
	case "number":
		return "scalar"
	default:
		return "table"
	}
}
