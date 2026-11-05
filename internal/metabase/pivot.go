package metabase

import (
	"fmt"
	"strings"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func applyPivotSettings(vs map[string]any, w parser.Widget, d *parser.Dashboard, reg *parser.Registry) {
	split := pivotColumnSplit(w, d, reg)
	vs["pivot_table.column_split"] = split

	if w.Options == nil {
		return
	}
	if rows, ok := w.Options["rows"]; ok && len(w.Rows) == 0 {
		vs["pivot.rows"] = rows
	}
	if cols, ok := w.Options["columns"]; ok && len(w.Columns) == 0 {
		vs["pivot.columns"] = cols
	}
	if truthy(w.Options["boldTotals"]) {
		vs["pivot.show_row_totals"] = true
		vs["pivot.show_column_totals"] = true
	}
	if neg, _ := w.Options["negatives"].(string); neg == "brackets" {
		settings, _ := vs["column_settings"].(map[string]any)
		if settings == nil {
			settings = map[string]any{}
		}
		for _, name := range widgetMeasures(w) {
			// ponytail: Metabase's formatter ignores negative_in_parentheses and still prints a leading minus.
			settings[`["name","`+name+`"]`] = map[string]any{"negative_in_parentheses": true}
		}
		vs["column_settings"] = settings
	}
	if rules, ok := w.Options["conditionalColors"].([]any); ok {
		var formatting []any
		for _, raw := range rules {
			rule, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			formatting = append(formatting, map[string]any{
				"columns":       []any{rule["measure"]},
				"type":          "single",
				"operator":      colorOperator(rule["when"]),
				"value":         colorValue(rule),
				"color":         rule["color"],
				"highlight_row": false,
			})
		}
		if len(formatting) > 0 {
			vs["table.column_formatting"] = formatting
		}
	}
}

func pivotColumnSplit(w parser.Widget, d *parser.Dashboard, reg *parser.Registry) map[string]any {
	index := map[string]int{}
	for i, name := range widgetMeasures(w) {
		index[name] = i
	}
	placed := map[string]bool{}
	rows := placePivot(w.Rows, index, placed, d, reg)
	cols := placePivot(w.Columns, index, placed, d, reg)
	values := []any{}
	for _, name := range widgetMeasures(w) {
		if placed[name] {
			continue
		}
		values = append(values, []any{"aggregation", index[name]})
	}
	return map[string]any{"rows": rows, "columns": cols, "values": values}
}

func placePivot(items []parser.PivotField, index map[string]int, placed map[string]bool, d *parser.Dashboard, reg *parser.Registry) []any {
	out := []any{}
	for _, item := range items {
		if item.Dimension != "" {
			out = append(out, columnName(d, reg, item.Dimension, true))
			continue
		}
		if item.Measure == "" {
			continue
		}
		placed[item.Measure] = true
		if i, ok := index[item.Measure]; ok {
			out = append(out, []any{"aggregation", i})
			continue
		}
		out = append(out, []any{"aggregation", item.Measure})
	}
	return out
}

// widgetMeasures is the card's aggregation order: declared measures, then any
// measure that appears only in a pivot row or column.
func widgetMeasures(w parser.Widget) []string {
	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, name := range w.Measures {
		add(name)
	}
	for _, item := range w.Rows {
		add(item.Measure)
	}
	for _, item := range w.Columns {
		add(item.Measure)
	}
	return names
}

func truthy(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func colorOperator(when any) string {
	switch when {
	case "positive":
		return ">"
	case "negative":
		return "<"
	default:
		if s, ok := when.(string); ok && s != "" {
			return s
		}
		return "<"
	}
}

func colorValue(rule map[string]any) any {
	if v, ok := rule["value"]; ok {
		return v
	}
	return 0
}

func buildTabs(d *parser.Dashboard) []map[string]any {
	if len(d.Spec.Tabs) == 0 {
		return nil
	}
	tabs := make([]map[string]any, 0, len(d.Spec.Tabs))
	for i, tab := range d.Spec.Tabs {
		name := tab.Title
		if tab.TitleTemplate != "" {
			name = resolveTitle(tab.TitleTemplate, d.Spec.Parameters)
		}
		item := map[string]any{
			"id":        i + 1,
			"name":      name,
			"position":  i,
			"entity_id": stableID("tab-" + tab.Name),
		}
		if tab.TitleTemplate != "" {
			item["title_template"] = tab.TitleTemplate
		}
		tabs = append(tabs, item)
	}
	return tabs
}

func dashboardTabID(d *parser.Dashboard, name string) int {
	if name == "" {
		return 0
	}
	for i, tab := range d.Spec.Tabs {
		if tab.Name == name {
			return i + 1
		}
	}
	return 0
}

func resolveTitle(tmpl string, params []parser.Parameter) string {
	out := tmpl
	for _, p := range params {
		if p.Default == nil {
			continue
		}
		out = strings.ReplaceAll(out, "{{"+p.Name+"}}", fmt.Sprint(p.Default))
	}
	return out
}
