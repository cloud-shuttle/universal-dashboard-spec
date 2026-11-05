package echarts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

// Emit returns a map of filename → TypeScript option object.
// Data is left as a placeholder so the caller can inject real series.
// Series type/axis, title templates, and parameters are included when present.
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
	title := w.Title
	if w.TitleTemplate != "" {
		title = w.TitleTemplate
	}

	opt := map[string]any{
		"title": map[string]any{"text": title},
		"tooltip": map[string]any{"trigger": tooltipTrigger(w.Type)},
	}

	// Parameters consumed by this widget
	if len(w.Parameters) > 0 {
		params := []map[string]any{}
		for _, name := range w.Parameters {
			for _, p := range d.Spec.Parameters {
				if p.Name == name {
					params = append(params, map[string]any{
						"name":   p.Name,
						"values": p.Values,
						"swaps":  p.Swaps,
					})
					break
				}
			}
		}
		if len(params) > 0 {
			opt["udsParameters"] = params
		}
	}

	series := buildSeries(w)

	switch w.Type {
	case "line", "area":
		opt["xAxis"] = map[string]any{"type": "category", "data": []any{"/* dimension values */"}}
		opt["yAxis"] = map[string]any{"type": "value"}
		opt["series"] = series
		if w.Type == "area" && len(series) > 0 {
			series[0]["areaStyle"] = map[string]any{}
		}

	case "bar", "stacked_bar":
		opt["xAxis"] = map[string]any{"type": "category", "data": []any{"/* dimension values */"}}
		opt["yAxis"] = map[string]any{"type": "value"}
		if w.Type == "stacked_bar" {
			for _, s := range series {
				s["stack"] = "total"
			}
		}
		opt["series"] = series

	case "combo":
		opt["xAxis"] = map[string]any{"type": "category", "data": []any{"/* dimension values */"}}
		opt["yAxis"] = []any{
			map[string]any{"type": "value", "position": "left"},
			map[string]any{"type": "value", "position": "right"},
		}
		opt["series"] = series

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
		opt["series"] = series

	case "number":
		opt["series"] = []any{
			map[string]any{
				"type": "gauge",
				"detail": map[string]any{"formatter": "{value}"},
				"data": []any{map[string]any{"value": 0, "name": title}},
			},
		}

	default:
		opt["series"] = series
	}

	// Goal line / markLine for deviation
	if goal, ok := w.Options["goalLine"]; ok {
		if s, ok := opt["series"].([]map[string]any); ok && len(s) > 0 {
			s[0]["markLine"] = map[string]any{
				"data": []any{map[string]any{"yAxis": goal}},
			}
		}
	}

	return opt
}

func buildSeries(w parser.Widget) []map[string]any {
	if len(w.Series) == 0 {
		// Fallback: one series from the widget type
		typ := w.Type
		if typ == "combo" || typ == "stacked_bar" || typ == "diverging" {
			typ = "bar"
		}
		return []map[string]any{{
			"name": w.Title,
			"type": typ,
			"data": []any{0},
		}}
	}

	var out []map[string]any
	for _, s := range w.Series {
		typ := s.Type
		if typ == "" {
			typ = "bar"
		}
		m := map[string]any{
			"name": s.Name,
			"type": typ,
			"data": []any{0},
		}
		if s.Color != "" {
			m["itemStyle"] = map[string]any{"color": s.Color}
		}
		if s.Axis == "right" {
			m["yAxisIndex"] = 1
		}
		if len(s.TooltipFields) > 0 {
			m["tooltipFields"] = s.TooltipFields
		}
		out = append(out, m)
	}
	return out
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

// silence unused import if strings is only used in future helpers
var _ = strings.TrimSpace
