package metabase

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

// EmitOptions lets the CLI override the source table declared in the UDS.
type EmitOptions struct {
	Database string
	Schema   string
	Table    string
}

// Emit produces a map of filename → YAML content suitable for Metabase Representations import.
// Entity IDs are deterministic (hash of name) so re-runs do not churn.
// Source table is taken from the UDS spec.source block, then overridden by opts if non-empty.
// Intent drives additional visualization and query settings (sort, stacking, goal line).
// If reg is non-nil, qualified measure/dimension names (model.name) are resolved against it.
// Filters and parameters declared on a widget are wired as parameter_mappings on the dashcard.
// titleTemplate, series colours, and radio controls are emitted where Metabase can express them.
func Emit(d *parser.Dashboard, opts EmitOptions, reg *parser.Registry) (map[string][]byte, error) {
	files := map[string][]byte{}

	db, schema, table := d.SourceOrDefault(
		defaultIfEmpty(opts.Database, "Sample Database"),
		defaultIfEmpty(opts.Schema, "PUBLIC"),
		defaultIfEmpty(opts.Table, "ORDERS"),
	)

	cardIDs := map[string]string{} // widget ID → entity ID

	for _, w := range d.Spec.Widgets {
		if w.Type == "text" || w.Type == "image" {
			continue
		}
		eid := stableID("card-" + w.ID)
		cardIDs[w.ID] = eid

		title := w.Title
		if w.TitleTemplate != "" {
			title = w.TitleTemplate // Metabase will resolve {{param}} if the param is present
		}

		card := map[string]any{
			"name":                   title,
			"entity_id":              eid,
			"display":                mapDisplay(w.Type),
			"type":                   "question",
			"query_type":             "query",
			"dataset_query":          buildMBQL(w, d, db, schema, table, reg),
			"visualization_settings": buildVizSettings(w),
			"serdes/meta": []map[string]string{{
				"id":    eid,
				"label": slug(w.Title),
				"model": "Card",
			}},
		}

		b, err := yaml.Marshal(card)
		if err != nil {
			return nil, err
		}
		files[fmt.Sprintf("collections/main/cards/%s_%s.yaml", eid, slug(w.Title))] = b
	}

	// Dashboard
	dashID := stableID("dashboard-" + d.Metadata.Name)
	dash := map[string]any{
		"name":        d.Metadata.Title,
		"description": d.Metadata.Description,
		"entity_id":   dashID,
		"parameters":  buildParameters(d),
		"dashcards":   buildDashcards(d, cardIDs),
		"serdes/meta": []map[string]string{{
			"id":    dashID,
			"label": slug(d.Metadata.Title),
			"model": "Dashboard",
		}},
	}

	b, err := yaml.Marshal(dash)
	if err != nil {
		return nil, err
	}
	files[fmt.Sprintf("collections/main/dashboards/%s_%s.yaml", dashID, slug(d.Metadata.Title))] = b

	return files, nil
}

func defaultIfEmpty(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func buildVizSettings(w parser.Widget) map[string]any {
	vs := map[string]any{}

	// deviation / goal line
	if goal, ok := w.Options["goalLine"]; ok {
		vs["graph.goal_value"] = goal
	}

	// part_to_whole → stacked
	if w.Intent == "part_to_whole" || w.Type == "stacked_bar" {
		vs["graph.stacking"] = "stacked"
	}

	if len(w.Dimensions) > 0 {
		vs["graph.dimensions"] = w.Dimensions
	}
	if len(w.Measures) > 0 {
		vs["graph.metrics"] = w.Measures
	}

	// series colours
	if len(w.Series) > 0 {
		colors := []string{}
		for _, s := range w.Series {
			if s.Color != "" {
				colors = append(colors, s.Color)
			}
		}
		if len(colors) > 0 {
			vs["graph.colors"] = colors
		}
	}

	// pivot formatting. Structured rows/columns keep measure and dimension order.
	// Metabase itself puts measures in values, so the mixed order is also stored
	// on pivot.row_order / pivot.column_order.
	if w.Type == "pivot" && (len(w.Rows) > 0 || len(w.Columns) > 0) {
		rowDims, rowOrder := splitPivot(w.Rows)
		colDims, colOrder := splitPivot(w.Columns)
		values := []string{}
		for _, item := range w.Rows {
			if item.Measure != "" {
				values = append(values, item.Measure)
			}
		}
		for _, item := range w.Columns {
			if item.Measure != "" {
				values = append(values, item.Measure)
			}
		}
		vs["pivot.rows"] = rowDims
		vs["pivot.columns"] = colDims
		vs["pivot.values"] = values
		vs["pivot.row_order"] = rowOrder
		vs["pivot.column_order"] = colOrder
	} else if w.Type == "pivot" {
		if rows, ok := w.Options["rows"]; ok {
			vs["pivot.rows"] = rows
		}
		if cols, ok := w.Options["columns"]; ok {
			vs["pivot.columns"] = cols
		}
	}

	return vs
}

func buildParameters(d *parser.Dashboard) []map[string]any {
	var params []map[string]any

	// Filters become dashboard parameters
	for _, f := range d.Spec.Filters {
		targetDim := f.Dimension
		if targetDim == "" && len(f.Bindings) > 0 {
			targetDim = bindingDimension(f.Bindings[0])
		}
		p := map[string]any{
			"id":     stableID("param-" + f.Name),
			"name":   f.Title,
			"slug":   f.Name,
			"type":   mapFilterType(f.Control),
			"target": []any{"dimension", []any{"template-tag", targetDim}},
		}
		if len(f.Values) > 0 {
			p["values_source_type"] = "static-list"
			p["values_source_config"] = map[string]any{"values": f.Values}
		}
		if f.Searchable {
			p["searchable"] = true
		}
		if len(f.Bindings) > 0 {
			p["bindings"] = f.Bindings
		}
		params = append(params, p)
	}

	// UDS parameters (user-set values) also become dashboard parameters
	for _, p := range d.Spec.Parameters {
		params = append(params, map[string]any{
			"id":      stableID("param-" + p.Name),
			"name":    p.Title,
			"slug":    p.Name,
			"type":    mapParamType(p.Type),
			"default": p.Default,
		})
	}

	return params
}

func mapFilterType(control string) string {
	switch control {
	case "radio":
		return "string/="
	case "multi_select":
		return "string/="
	case "date_range":
		return "date/range"
	case "date_picker":
		return "date/single"
	default:
		return "string/="
	}
}

func mapParamType(t string) string {
	switch t {
	case "number":
		return "number/="
	case "date":
		return "date/single"
	case "date_range":
		return "date/range"
	case "boolean":
		return "string/="
	default:
		return "string/="
	}
}

func buildDashcards(d *parser.Dashboard, cardIDs map[string]string) []map[string]any {
	var cards []map[string]any
	for i, w := range d.Spec.Widgets {
		if w.Type == "text" || w.Type == "image" {
			body := w.Content
			if w.Type == "image" {
				body = w.Src
			}
			cards = append(cards, map[string]any{
				"id":      i + 1,
				"card_id": nil,
				"row":     w.Layout.Y,
				"col":     w.Layout.X,
				"size_x":  w.Layout.W,
				"size_y":  w.Layout.H,
				"visualization_settings": map[string]any{
					"text": body,
					"virtual_card": map[string]any{
						"display":  "text",
						"archived": false,
					},
				},
			})
			continue
		}
		mappings := buildParameterMappings(d, w)
		cards = append(cards, map[string]any{
			"id":                 i + 1,
			"card_id":            cardIDs[w.ID],
			"row":                w.Layout.Y,
			"col":                w.Layout.X,
			"size_x":             w.Layout.W,
			"size_y":             w.Layout.H,
			"entity_id":          stableID("dashcard-" + w.ID),
			"parameter_mappings": mappings,
		})
	}
	return cards
}

// buildParameterMappings wires each filter and parameter listed on the widget
// to the corresponding dashboard parameter.
func buildParameterMappings(d *parser.Dashboard, w parser.Widget) []map[string]any {
	var mappings []map[string]any

	for _, filterName := range w.Filters {
		var f *parser.Filter
		for i := range d.Spec.Filters {
			if d.Spec.Filters[i].Name == filterName {
				f = &d.Spec.Filters[i]
				break
			}
		}
		if f == nil {
			continue
		}
		mappings = append(mappings, map[string]any{
			"parameter_id": stableID("param-" + f.Name),
			"card_id":      stableID("card-" + w.ID),
			"target": []any{
				"dimension",
				[]any{"template-tag", filterDimensionForWidget(*f, w)},
			},
		})
	}

	for _, paramName := range w.Parameters {
		mappings = append(mappings, map[string]any{
			"parameter_id": stableID("param-" + paramName),
			"card_id":      stableID("card-" + w.ID),
			"target": []any{
				"variable",
				[]any{"template-tag", paramName},
			},
		})
	}

	return mappings
}

func mapDisplay(t string) string {
	switch t {
	case "line":
		return "line"
	case "bar", "stacked_bar", "diverging":
		return "bar"
	case "pie":
		return "pie"
	case "table", "pivot":
		return "table"
	case "number":
		return "scalar"
	case "combo":
		return "combo"
	default:
		return "table"
	}
}

func splitPivot(items []parser.PivotField) (dims []string, order []map[string]string) {
	dims = []string{}
	order = []map[string]string{}
	for _, item := range items {
		if item.Dimension != "" {
			dims = append(dims, item.Dimension)
			order = append(order, map[string]string{"dimension": item.Dimension})
			continue
		}
		if item.Measure != "" {
			order = append(order, map[string]string{"measure": item.Measure})
		}
	}
	return dims, order
}

func bindingDimension(binding string) string {
	if i := strings.LastIndex(binding, "."); i >= 0 {
		return binding[i+1:]
	}
	return binding
}

// filterDimensionForWidget picks the binding whose model matches a measure or
// dimension on the widget. A single dimension binding is returned as-is.
func filterDimensionForWidget(f parser.Filter, w parser.Widget) string {
	if len(f.Bindings) == 0 {
		return f.Dimension
	}
	models := map[string]bool{}
	for _, n := range w.Measures {
		if i := strings.IndexByte(n, '.'); i > 0 {
			models[n[:i]] = true
		}
	}
	for _, n := range w.Dimensions {
		if i := strings.IndexByte(n, '.'); i > 0 {
			models[n[:i]] = true
		}
	}
	var plain string
	for _, b := range f.Bindings {
		if i := strings.IndexByte(b, '.'); i > 0 {
			if models[b[:i]] {
				return b[i+1:]
			}
			continue
		}
		if plain == "" {
			plain = b
		}
	}
	if plain != "" {
		return plain
	}
	return bindingDimension(f.Bindings[0])
}

func findMeasure(d *parser.Dashboard, name string, reg *parser.Registry) *parser.Measure {
	if reg != nil {
		parts := strings.SplitN(name, ".", 2)
		if len(parts) == 2 {
			if m, ok := reg.Get(parts[0]); ok {
				for i := range m.Spec.Measures {
					if m.Spec.Measures[i].Name == parts[1] {
						return &m.Spec.Measures[i]
					}
				}
			}
		}
	}
	for i := range d.Spec.Measures {
		if d.Spec.Measures[i].Name == name {
			return &d.Spec.Measures[i]
		}
	}
	return nil
}

func findDimension(d *parser.Dashboard, name string, reg *parser.Registry) *parser.Dimension {
	if reg != nil {
		parts := strings.SplitN(name, ".", 2)
		if len(parts) == 2 {
			if m, ok := reg.Get(parts[0]); ok {
				for i := range m.Spec.Dimensions {
					if m.Spec.Dimensions[i].Name == parts[1] {
						return &m.Spec.Dimensions[i]
					}
				}
			}
		}
	}
	for i := range d.Spec.Dimensions {
		if d.Spec.Dimensions[i].Name == name {
			return &d.Spec.Dimensions[i]
		}
	}
	return nil
}

// stableID produces a 21-character NanoID-like string that is deterministic
// for a given input. Metabase entity IDs are 21 chars.
func stableID(s string) string {
	h := sha256.Sum256([]byte(s))
	// base64.RawURLEncoding is URL-safe and has no padding
	id := base64.RawURLEncoding.EncodeToString(h[:16])
	if len(id) > 21 {
		id = id[:21]
	}
	// NanoIDs use A-Za-z0-9_- ; RawURLEncoding already does
	return id
}

func slug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
