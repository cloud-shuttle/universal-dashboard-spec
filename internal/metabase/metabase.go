package metabase

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

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
// Each card uses the source of its measure model. -database overrides the base table and every join. -schema and -table override the base table.
// A query field from another model is a left join on a shared entity, and one side must be a primary key. A filter binding does not add a join.
// Intent drives additional visualization and query settings (sort, stacking, goal line).
// If reg is non-nil, qualified measure/dimension names (model.name) are resolved against it.
// Filters declared on a widget are wired as field targets on the dashcard.
// titleTemplate is substituted from parameter defaults. Series colours and radio controls are emitted where Metabase can express them.
func Emit(d *parser.Dashboard, opts EmitOptions, reg *parser.Registry) (map[string][]byte, error) {
	files := map[string][]byte{}

	cardIDs := map[string]string{} // widget ID → entity ID
	sources := map[string]cardSource{}
	var sourceWarns []string

	for _, w := range d.Spec.Widgets {
		if w.Type == "text" || w.Type == "image" {
			continue
		}
		src, err := resolveCardSource(w, d, opts, reg)
		if err != nil {
			return nil, fmt.Errorf("widget %s: %w", w.ID, err)
		}
		sourceWarns = append(sourceWarns, src.warnings...)
		sources[w.ID] = src
		query, err := buildMBQL(w, d, src, reg)
		if err != nil {
			return nil, fmt.Errorf("widget %s: %w", w.ID, err)
		}
		eid := stableID("card-" + w.ID)
		cardIDs[w.ID] = eid

		title := w.Title
		if w.TitleTemplate != "" {
			title = resolveTitle(w.TitleTemplate, d.Spec.Parameters)
		}

		card := map[string]any{
			"name":                   title,
			"entity_id":              eid,
			"display":                mapDisplay(w.Type),
			"type":                   "question",
			"query_type":             "query",
			"dataset_query":          query,
			"visualization_settings": buildVizSettings(w, d, reg),
			"serdes/meta": []map[string]string{{
				"id":    eid,
				"label": slug(title),
				"model": "Card",
			}},
		}

		b, err := marshalDoc(card)
		if err != nil {
			return nil, err
		}
		files[fmt.Sprintf("collections/main/cards/%s_%s.yaml", eid, slug(title))] = b
	}
	for _, msg := range uniqueStrings(sourceWarns) {
		fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
	}

	// Dashboard
	dashID := stableID("dashboard-" + d.Metadata.Name)
	dash := map[string]any{
		"name":        d.Metadata.Title,
		"description": d.Metadata.Description,
		"entity_id":   dashID,
		"parameters":  buildParameters(d),
		"dashcards":   buildDashcards(d, cardIDs, sources, reg),
		"serdes/meta": []map[string]string{{
			"id":    dashID,
			"label": slug(d.Metadata.Title),
			"model": "Dashboard",
		}},
	}
	if tabs := buildTabs(d); len(tabs) > 0 {
		dash["tabs"] = tabs
	}
	for _, msg := range lossWarnings(d) {
		fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
	}

	b, err := marshalDoc(dash)
	if err != nil {
		return nil, err
	}
	files[fmt.Sprintf("collections/main/dashboards/%s_%s.yaml", dashID, slug(d.Metadata.Title))] = b

	return files, nil
}

func buildVizSettings(w parser.Widget, d *parser.Dashboard, reg *parser.Registry) map[string]any {
	vs := map[string]any{}

	// deviation / goal line
	if goal, ok := w.Options["goalLine"]; ok {
		vs["graph.goal_value"] = goal
	}

	// part_to_whole → stacked
	if w.Intent == "part_to_whole" || w.Type == "stacked_bar" {
		vs["graph.stacking"] = "stacked"
	}

	if dims := resultNames(w.Dimensions, true, d, reg); len(dims) > 0 {
		vs["graph.dimensions"] = dims
	}
	if metrics := resultNames(w.Measures, false, d, reg); len(metrics) > 0 {
		vs["graph.metrics"] = metrics
	}

	// series_settings carries per-series display, axis, and colour.
	// graph.colors keeps the same colour list for readers that only look there.
	if len(w.Series) > 0 {
		colors := []string{}
		settings := map[string]any{}
		autoSplit := false
		for _, s := range w.Series {
			one := map[string]any{}
			if s.Color != "" {
				colors = append(colors, s.Color)
				one["color"] = s.Color
			}
			if s.Type != "" {
				one["display"] = s.Type
			}
			if s.Axis != "" {
				one["axis"] = s.Axis
				if s.Axis == "right" {
					autoSplit = true
				}
			}
			if len(one) > 0 {
				settings[shortName(s.Name)] = one
			}
		}
		if tips := tooltipColumns(w.Series); len(tips) > 0 {
			vs["graph.tooltip_columns"] = tips
		}
		if len(colors) > 0 {
			vs["graph.colors"] = colors
		}
		if len(settings) > 0 {
			vs["series_settings"] = settings
		}
		if autoSplit {
			vs["graph.y_axis.auto_split"] = true
		}
	}

	if w.Type == "pivot" {
		applyPivotSettings(vs, w, d, reg)
	}

	return vs
}

func buildParameters(d *parser.Dashboard) []map[string]any {
	var params []map[string]any

	// Filters become dashboard parameters. The field target lives on each dashcard.
	for _, f := range d.Spec.Filters {
		p := map[string]any{
			"id":   stableID("param-" + f.Name),
			"name": f.Title,
			"slug": f.Name,
			"type": mapFilterType(f.Control),
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

func buildDashcards(d *parser.Dashboard, cardIDs map[string]string, sources map[string]cardSource, reg *parser.Registry) []map[string]any {
	var cards []map[string]any
	for i, w := range d.Spec.Widgets {
		if w.Type == "text" || w.Type == "image" {
			body := w.Content
			if w.Type == "image" {
				body = w.Src
			}
			card := map[string]any{
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
			}
			if tabID := dashboardTabID(d, w.Tab); tabID != 0 {
				card["dashboard_tab_id"] = tabID
			}
			cards = append(cards, card)
			continue
		}
		mappings, warns := buildParameterMappings(d, w, sources[w.ID], reg)
		for _, msg := range warns {
			fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
		}
		card := map[string]any{
			"id":                 i + 1,
			"card_id":            cardIDs[w.ID],
			"row":                w.Layout.Y,
			"col":                w.Layout.X,
			"size_x":             w.Layout.W,
			"size_y":             w.Layout.H,
			"entity_id":          stableID("dashcard-" + w.ID),
			"parameter_mappings": mappings,
		}
		if tabID := dashboardTabID(d, w.Tab); tabID != 0 {
			card["dashboard_tab_id"] = tabID
		}
		cards = append(cards, card)
	}
	return cards
}

// buildParameterMappings wires each filter on the widget to a field on the card.
// A function-source argument is a template tag instead. Other widget parameters only substitute title templates.
func buildParameterMappings(d *parser.Dashboard, w parser.Widget, src cardSource, reg *parser.Registry) ([]map[string]any, []string) {
	var mappings []map[string]any
	var warnings []string

	for _, filterName := range w.Filters {
		f := findFilter(d, filterName)
		if f == nil {
			warnings = append(warnings, fmt.Sprintf("widget %q: filter %q is not declared, mapping skipped", w.ID, filterName))
			continue
		}
		if arg, ok := src.argumentForFilter(f.Name); ok {
			mappings = append(mappings, templateMapping(f.Name, w.ID, arg.Name))
			continue
		}
		binding := filterBindingForWidget(*f, w)
		model, _, qualified := splitRef(binding)
		if qualified && model != src.base {
			if _, ok := src.joined[model]; !ok {
				warnings = append(warnings, fmt.Sprintf("widget %q: filter %q binding %q is not on this card, mapping skipped", w.ID, f.Name, binding))
				continue
			}
		}
		mappings = append(mappings, map[string]any{
			"parameter_id": stableID("param-" + f.Name),
			"card_id":      stableID("card-" + w.ID),
			"target": []any{
				"dimension",
				src.field(model, columnName(d, reg, binding, true), nil),
			},
		})
	}
	for _, name := range w.Parameters {
		arg, ok := src.argumentForParameter(name)
		if !ok {
			continue
		}
		mappings = append(mappings, templateMapping(name, w.ID, arg.Name))
	}

	return mappings, warnings
}

func templateMapping(control, widgetID, argName string) map[string]any {
	return map[string]any{
		"parameter_id": stableID("param-" + control),
		"card_id":      stableID("card-" + widgetID),
		"target":       []any{"variable", []any{"template-tag", argName}},
	}
}

// lossWarnings reports query behaviour the Metabase emitter does not carry.
// Parameter swaps stay on the control. The card keeps the measures written on
// the widget. Drill actions have no dashcard click behaviour yet.
func lossWarnings(d *parser.Dashboard) []string {
	var out []string
	for _, p := range d.Spec.Parameters {
		if len(p.Swaps) > 0 {
			out = append(out, fmt.Sprintf("parameter %q swaps are not applied to the query; cards keep their listed measures", p.Name))
		}
	}
	for _, w := range d.Spec.Widgets {
		for _, a := range w.Actions {
			out = append(out, fmt.Sprintf("widget %q: action %q is not emitted", w.ID, a.Type))
		}
		if w.VisibleWhen != nil && w.VisibleWhen.Parameter != "" {
			out = append(out, fmt.Sprintf("widget %q is visible when %s equals %v; Metabase shows the card for every value", w.ID, w.VisibleWhen.Parameter, w.VisibleWhen.Equals))
		}
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, msg := range in {
		if seen[msg] {
			continue
		}
		seen[msg] = true
		out = append(out, msg)
	}
	return out
}

func tooltipColumns(series []parser.Series) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range series {
		for _, name := range s.TooltipFields {
			short := shortName(name)
			if short == "" || seen[short] {
				continue
			}
			seen[short] = true
			out = append(out, short)
		}
	}
	return out
}

func mapDisplay(t string) string {
	switch t {
	case "line":
		return "line"
	case "bar", "stacked_bar", "diverging":
		return "bar"
	case "pie":
		return "pie"
	case "table":
		return "table"
	case "pivot":
		return "pivot"
	case "number":
		return "scalar"
	case "combo":
		return "combo"
	default:
		return "table"
	}
}

// filterBindingForWidget picks the binding whose model matches a measure or
// dimension on the widget. The model prefix is kept. A binding whose model is
// not on the card is left unmapped, because a filter does not add a join.
func filterBindingForWidget(f parser.Filter, w parser.Widget) string {
	if len(f.Bindings) == 0 {
		return f.Dimension
	}
	models := map[string]bool{}
	for _, n := range append(append([]string{}, w.Measures...), w.Dimensions...) {
		if model, _, ok := splitRef(n); ok {
			models[model] = true
		}
	}
	var plain string
	for _, b := range f.Bindings {
		model, _, ok := splitRef(b)
		if !ok {
			if plain == "" {
				plain = b
			}
			continue
		}
		if models[model] {
			return b
		}
	}
	if plain != "" {
		return plain
	}
	return f.Bindings[0]
}

func findFilter(d *parser.Dashboard, name string) *parser.Filter {
	for i := range d.Spec.Filters {
		if d.Spec.Filters[i].Name == name {
			return &d.Spec.Filters[i]
		}
	}
	return nil
}

func resultNames(names []string, dim bool, d *parser.Dashboard, reg *parser.Registry) []string {
	if len(names) == 0 {
		return nil
	}
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = columnName(d, reg, name, dim)
	}
	return out
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
