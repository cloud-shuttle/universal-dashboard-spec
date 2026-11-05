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
func Emit(d *parser.Dashboard, opts EmitOptions) (map[string][]byte, error) {
	files := map[string][]byte{}

	db, schema, table := d.SourceOrDefault(
		defaultIfEmpty(opts.Database, "Sample Database"),
		defaultIfEmpty(opts.Schema, "PUBLIC"),
		defaultIfEmpty(opts.Table, "ORDERS"),
	)

	cardIDs := map[string]string{} // widget ID → entity ID

	for _, w := range d.Spec.Widgets {
		eid := stableID("card-" + w.ID)
		cardIDs[w.ID] = eid

		card := map[string]any{
			"name":        w.Title,
			"entity_id":   eid,
			"display":     mapDisplay(w.Type),
			"type":        "question",
			"query_type":  "query",
			"dataset_query": buildMBQL(w, d, db, schema, table),
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

func buildMBQL(w parser.Widget, d *parser.Dashboard, db, schema, table string) map[string]any {
	agg := []any{}
	for _, mName := range w.Measures {
		m := findMeasure(d, mName)
		if m == nil {
			continue
		}
		agg = append(agg, []any{m.Aggregation, fieldRef(db, schema, table, m.Field)})
	}

	breakout := []any{}
	for _, dimName := range w.Dimensions {
		dim := findDimension(d, dimName)
		if dim == nil {
			continue
		}
		ref := fieldRef(db, schema, table, dim.Field)
		if dim.TemporalGrain != "" {
			ref = []any{"field", ref[1], map[string]any{"temporal-unit": dim.TemporalGrain}}
		}
		breakout = append(breakout, ref)
	}

	query := map[string]any{
		"source-table": []any{db, schema, table},
		"aggregation":  agg,
		"breakout":     breakout,
	}

	// ranking → order by first measure descending
	if w.Intent == "ranking" && len(agg) > 0 {
		query["order-by"] = []any{
			[]any{"desc", []any{"aggregation", 0}},
		}
	}

	return map[string]any{
		"database": db,
		"type":     "query",
		"query":    query,
	}
}

func fieldRef(db, schema, table, field string) []any {
	return []any{"field", []any{db, schema, table, field}, nil}
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
	return vs
}

func buildParameters(d *parser.Dashboard) []map[string]any {
	var params []map[string]any
	for _, f := range d.Spec.Filters {
		params = append(params, map[string]any{
			"id":     stableID("param-" + f.Name),
			"name":   f.Title,
			"slug":   f.Name,
			"type":   "string/=",
			"target": []any{"dimension", []any{"template-tag", f.Dimension}},
		})
	}
	return params
}

func buildDashcards(d *parser.Dashboard, cardIDs map[string]string) []map[string]any {
	var cards []map[string]any
	for i, w := range d.Spec.Widgets {
		cards = append(cards, map[string]any{
			"id":         i + 1,
			"card_id":    cardIDs[w.ID],
			"row":        w.Layout.Y,
			"col":        w.Layout.X,
			"size_x":     w.Layout.W,
			"size_y":     w.Layout.H,
			"entity_id":  stableID("dashcard-" + w.ID),
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

func findMeasure(d *parser.Dashboard, name string) *parser.Measure {
	for i := range d.Spec.Measures {
		if d.Spec.Measures[i].Name == name {
			return &d.Spec.Measures[i]
		}
	}
	return nil
}

func findDimension(d *parser.Dashboard, name string) *parser.Dimension {
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
