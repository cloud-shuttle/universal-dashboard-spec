package echarts

import (
	"encoding/json"
	"fmt"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

// Emit returns a map of filename → TypeScript option object.
// Data is left as a placeholder so the caller can inject real series.
func Emit(d *parser.Dashboard) (map[string][]byte, error) {
	files := map[string][]byte{}

	for _, w := range d.Spec.Widgets {
		opt := buildOption(w, d)
		b, err := json.MarshalIndent(opt, "", "  ")
		if err != nil {
			return nil, err
		}
		// Wrap as a TypeScript const
		ts := fmt.Sprintf("export const %s = %s as const;\n", camel(w.ID), string(b))
		files[w.ID+".ts"] = []byte(ts)
	}

	return files, nil
}

func buildOption(w parser.Widget, d *parser.Dashboard) map[string]any {
	opt := map[string]any{
		"title": map[string]any{"text": w.Title},
		"tooltip": map[string]any{"trigger": tooltipTrigger(w.Type)},
	}

	switch w.Type {
	case "line", "area":
		opt["xAxis"] = map[string]any{"type": "category", "data": []any{"/* dimension values */"}}
		opt["yAxis"] = map[string]any{"type": "value"}
		opt["series"] = []any{seriesFor(w, "line")}
		if w.Type == "area" {
			opt["series"].([]any)[0].(map[string]any)["areaStyle"] = map[string]any{}
		}

	case "bar", "stacked_bar":
		opt["xAxis"] = map[string]any{"type": "category", "data": []any{"/* dimension values */"}}
		opt["yAxis"] = map[string]any{"type": "value"}
		s := seriesFor(w, "bar")
		if w.Type == "stacked_bar" {
			s["stack"] = "total"
		}
		opt["series"] = []any{s}

	case "pie":
		opt["series"] = []any{
			map[string]any{
				"type": "pie",
				"data": []any{map[string]any{"name": "/* category */", "value": 0}},
			},
		}

	case "scatter":
		opt["xAxis"] = map[string]any{"type": "value"}
		opt["yAxis"] = map[string]any{"type": "value"}
		opt["series"] = []any{seriesFor(w, "scatter")}

	case "number":
		opt["series"] = []any{
			map[string]any{
				"type": "gauge",
				"detail": map[string]any{"formatter": "{value}"},
				"data": []any{map[string]any{"value": 0, "name": w.Title}},
			},
		}

	default:
		opt["series"] = []any{seriesFor(w, "bar")}
	}

	// Goal line / markLine for deviation
	if goal, ok := w.Options["goalLine"]; ok {
		if series, ok := opt["series"].([]any); ok && len(series) > 0 {
			if s, ok := series[0].(map[string]any); ok {
				s["markLine"] = map[string]any{
					"data": []any{map[string]any{"yAxis": goal}},
				}
			}
		}
	}

	return opt
}

func seriesFor(w parser.Widget, typ string) map[string]any {
	return map[string]any{
		"name": w.Title,
		"type": typ,
		"data": []any{0}, // placeholder — inject real data
	}
}

func tooltipTrigger(t string) string {
	switch t {
	case "pie":
		return "item"
	default:
		return "axis"
	}
}

func camel(s string) string {
	if s == "" {
		return s
	}
	// simple: revenue_over_time → revenueOverTime
	out := []rune{}
	upper := false
	for _, r := range s {
		if r == '_' || r == '-' {
			upper = true
			continue
		}
		if upper {
			if r >= 'a' && r <= 'z' {
				r = r - 'a' + 'A'
			}
			upper = false
		}
		out = append(out, r)
	}
	return string(out)
}
