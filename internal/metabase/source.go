package metabase

import (
	"fmt"
	"strings"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

// cardSource is the physical table for one card, plus left joins for any
// other semantic model the card references.
type cardSource struct {
	db, schema, table string
	base              string
	joins             []mbqlJoin
	joined            map[string]mbqlJoin
}

type mbqlJoin struct {
	alias             string
	db, schema, table string
	left, right       string
}

func resolveCardSource(w parser.Widget, d *parser.Dashboard, opts EmitOptions, reg *parser.Registry) (cardSource, error) {
	base, others := cardModels(w, d, reg)
	src := cardSource{
		base:   base,
		joined: map[string]mbqlJoin{},
	}
	src.db, src.schema, src.table = tableOf(modelByName(reg, base), d, opts, true)
	for _, name := range others {
		other := modelByName(reg, name)
		if other == nil {
			continue
		}
		baseModel := modelByName(reg, base)
		if baseModel == nil {
			return cardSource{}, fmt.Errorf("dimension from %q needs a join, and the card has no model to join from", name)
		}
		left, right, ok := matchEntities(baseModel, other)
		if !ok {
			return cardSource{}, fmt.Errorf("cannot join %q to %q: no shared entity", base, name)
		}
		db, schema, table := tableOf(other, d, opts, false)
		j := mbqlJoin{alias: name, db: db, schema: schema, table: table, left: left, right: right}
		src.joins = append(src.joins, j)
		src.joined[name] = j
	}
	return src, nil
}

func (s cardSource) field(model, field string, extra map[string]any) []any {
	db, schema, table := s.db, s.schema, s.table
	opts := map[string]any{}
	if model != "" && model != s.base {
		if j, ok := s.joined[model]; ok {
			db, schema, table = j.db, j.schema, j.table
			opts["join-alias"] = j.alias
		}
	}
	for k, v := range extra {
		opts[k] = v
	}
	var opt any
	if len(opts) > 0 {
		opt = opts
	}
	return []any{"field", []any{db, schema, table, field}, opt}
}

func (s cardSource) joinClauses() []any {
	out := make([]any, 0, len(s.joins))
	for _, j := range s.joins {
		out = append(out, map[string]any{
			"source-table": []any{j.db, j.schema, j.table},
			"alias":        j.alias,
			"strategy":     "left-join",
			"condition": []any{
				"=",
				[]any{"field", []any{s.db, s.schema, s.table, j.left}, nil},
				[]any{"field", []any{j.db, j.schema, j.table, j.right}, map[string]any{"join-alias": j.alias}},
			},
		})
	}
	return out
}

// tableOf resolves a physical table. CLI overrides apply to the base table only.
func tableOf(m *parser.SemanticModel, d *parser.Dashboard, opts EmitOptions, base bool) (string, string, string) {
	db, schema, table := "", "", ""
	if m != nil && m.Spec.Source != nil {
		db, schema, table = m.Spec.Source.Database, m.Spec.Source.Schema, m.Spec.Source.Table
	}
	if d.Spec.Source != nil {
		if db == "" {
			db = d.Spec.Source.Database
		}
		if schema == "" {
			schema = d.Spec.Source.Schema
		}
		if table == "" {
			table = d.Spec.Source.Table
		}
	}
	if db == "" {
		db = "Sample Database"
	}
	if schema == "" {
		schema = "PUBLIC"
	}
	if table == "" {
		table = "ORDERS"
	}
	if base {
		if opts.Database != "" {
			db = opts.Database
		}
		if opts.Schema != "" {
			schema = opts.Schema
		}
		if opts.Table != "" {
			table = opts.Table
		}
	}
	return db, schema, table
}

func cardModels(w parser.Widget, d *parser.Dashboard, reg *parser.Registry) (base string, others []string) {
	seen := map[string]bool{}
	var order []string
	add := func(name string) {
		model, _, ok := splitRef(name)
		if !ok || seen[model] || reg == nil {
			return
		}
		if _, ok := reg.Get(model); !ok {
			return
		}
		seen[model] = true
		order = append(order, model)
	}
	for _, name := range widgetMeasures(w) {
		add(name)
	}
	for _, name := range w.Dimensions {
		add(name)
	}
	for _, item := range append(append([]parser.PivotField{}, w.Rows...), w.Columns...) {
		add(item.Dimension)
		add(item.Measure)
	}
	for _, p := range w.FixedFilters {
		add(p.Dimension)
	}
	for _, name := range w.Filters {
		f := findFilter(d, name)
		if f == nil {
			continue
		}
		add(filterBindingForWidget(*f, w))
	}
	for _, name := range widgetMeasures(w) {
		m, _ := lookupMeasure(d, name, reg)
		if m == nil {
			continue
		}
		for _, p := range m.Where {
			add(p.Dimension)
		}
	}
	if len(order) == 0 {
		return "", nil
	}
	return order[0], order[1:]
}

func matchEntities(base, other *parser.SemanticModel) (left, right string, ok bool) {
	type ent struct{ typ, field string }
	otherBy := map[string]ent{}
	for _, e := range other.Spec.Entities {
		otherBy[e.Name] = ent{e.Type, e.Field}
	}
	var fbL, fbR string
	for _, e := range base.Spec.Entities {
		o, found := otherBy[e.Name]
		if !found || e.Field == "" || o.field == "" {
			continue
		}
		if e.Type == "foreign" && o.typ == "primary" {
			return e.Field, o.field, true
		}
		if fbL == "" {
			fbL, fbR = e.Field, o.field
		}
	}
	if fbL != "" {
		return fbL, fbR, true
	}
	return "", "", false
}

func modelByName(reg *parser.Registry, name string) *parser.SemanticModel {
	if reg == nil || name == "" {
		return nil
	}
	m, ok := reg.Get(name)
	if !ok {
		return nil
	}
	return m
}

func splitRef(name string) (model, local string, qualified bool) {
	model, local, ok := strings.Cut(name, ".")
	if !ok || model == "" || local == "" {
		return "", name, false
	}
	return model, local, true
}

func shortName(name string) string {
	_, local, ok := splitRef(name)
	if ok {
		return local
	}
	return name
}

// lookupMeasure resolves a measure, or a simple/ltm metric, to a measure.
// A type:ltm metric becomes an ltm aggregation of the measure it names.
func lookupMeasure(d *parser.Dashboard, name string, reg *parser.Registry) (*parser.Measure, string) {
	model, _, _ := splitRef(name)
	if m := findMeasure(d, name, reg); m != nil {
		return m, model
	}
	if reg == nil || model == "" {
		return nil, model
	}
	sm, ok := reg.Get(model)
	if !ok {
		return nil, model
	}
	local := shortName(name)
	for i := range sm.Spec.Metrics {
		met := sm.Spec.Metrics[i]
		if met.Name != local || met.Measure == "" {
			continue
		}
		if met.Type != "ltm" && met.Type != "simple" {
			continue
		}
		base := findMeasure(d, model+"."+met.Measure, reg)
		if base == nil {
			return nil, model
		}
		cp := *base
		cp.Name = met.Name
		if met.Type == "ltm" && cp.Expression == "" {
			cp.Aggregation = "ltm"
		}
		return &cp, model
	}
	return nil, model
}

func columnName(d *parser.Dashboard, reg *parser.Registry, ref string, dim bool) string {
	if dim {
		if found := findDimension(d, ref, reg); found != nil && found.Field != "" {
			return found.Field
		}
	}
	return shortName(ref)
}
