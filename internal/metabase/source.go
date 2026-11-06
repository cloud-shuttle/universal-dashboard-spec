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
	function          string
	arguments         parser.Arguments
	base              string
	joins             []mbqlJoin
	joined            map[string]mbqlJoin
	warnings          []string
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
	var warns []string
	baseModel := modelByName(reg, base)
	src.db, src.schema, src.table, warns = tableOf(baseModel, d, opts, true)
	if baseModel != nil && baseModel.Spec.Source != nil && baseModel.Spec.Source.Function != "" {
		src.function = baseModel.Spec.Source.Function
		src.arguments = baseModel.Spec.Source.Arguments
		warns = dropTableWarnings(warns)
	}
	src.warnings = append(src.warnings, warns...)
	if src.function != "" && len(others) > 0 {
		return cardSource{}, fmt.Errorf("cannot join onto function source %q", src.function)
	}
	for _, name := range others {
		other := modelByName(reg, name)
		if other == nil {
			continue
		}
		if baseModel == nil {
			return cardSource{}, fmt.Errorf("dimension from %q needs a join, and the card has no model to join from", name)
		}
		left, right, err := matchEntities(baseModel, other)
		if err != nil {
			return cardSource{}, fmt.Errorf("cannot join %q to %q: %w", base, name, err)
		}
		db, schema, table, warns := tableOf(other, d, opts, false)
		src.warnings = append(src.warnings, warns...)
		j := mbqlJoin{alias: name, db: db, schema: schema, table: table, left: left, right: right}
		src.joins = append(src.joins, j)
		src.joined[name] = j
	}
	return src, nil
}

func dropTableWarnings(in []string) []string {
	var out []string
	for _, msg := range in {
		if strings.Contains(msg, "has no table") {
			continue
		}
		out = append(out, msg)
	}
	return out
}

func (s cardSource) field(model, field string, extra map[string]any) []any {
	if s.function != "" && (model == "" || model == s.base) {
		opts := map[string]any{"base-type": "type/*"}
		for k, v := range extra {
			opts[k] = v
		}
		return []any{"field", field, opts}
	}
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

// tableOf resolves a physical table. -database applies to the base table and to
// every join. -schema and -table apply to the base table only.
func tableOf(m *parser.SemanticModel, d *parser.Dashboard, opts EmitOptions, base bool) (string, string, string, []string) {
	db, schema, table := "", "", ""
	modelDB, modelTable := "", ""
	hasSource := m != nil && m.Spec.Source != nil
	if hasSource {
		db, schema, table = m.Spec.Source.Database, m.Spec.Source.Schema, m.Spec.Source.Table
		modelDB, modelTable = db, table
	}
	dashDB, dashTable := "", ""
	if d.Spec.Source != nil {
		dashDB, dashTable = d.Spec.Source.Database, d.Spec.Source.Table
		if db == "" {
			db = dashDB
		}
		if schema == "" {
			schema = d.Spec.Source.Schema
		}
		if table == "" {
			table = dashTable
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
	if opts.Database != "" {
		db = opts.Database
	}
	if base {
		if opts.Schema != "" {
			schema = opts.Schema
		}
		if opts.Table != "" {
			table = opts.Table
		}
	}
	var warnings []string
	if m != nil {
		name := m.Metadata.Name
		switch {
		case !hasSource:
			warnings = append(warnings, fmt.Sprintf("model %q has no source; using %s.%s.%s", name, db, schema, table))
		default:
			if modelDB == "" && dashDB == "" && opts.Database == "" {
				warnings = append(warnings, fmt.Sprintf("model %q has no database; using %q", name, db))
			}
			cliTable := base && opts.Table != ""
			if modelTable == "" && dashTable == "" && !cliTable {
				warnings = append(warnings, fmt.Sprintf("model %q has no table; using %q", name, table))
			}
		}
	}
	return db, schema, table, warnings
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

// matchEntities joins on a shared entity only when one side is a primary key.
// Foreign key on the base and primary key on the other model is preferred.
// The reverse is accepted. Two foreign keys are refused: customer = customer
// multiplies every fact row.
func matchEntities(base, other *parser.SemanticModel) (left, right string, err error) {
	type ent struct{ typ, field string }
	otherBy := map[string]ent{}
	for _, e := range other.Spec.Entities {
		otherBy[e.Name] = ent{e.Type, e.Field}
	}
	var bestL, bestR string
	bestRank := 0
	var nonUnique string
	for _, e := range base.Spec.Entities {
		o, found := otherBy[e.Name]
		if !found || e.Field == "" || o.field == "" {
			continue
		}
		rank := 0
		switch {
		case e.Type == "foreign" && o.typ == "primary":
			rank = 3
		case e.Type == "primary" && o.typ == "foreign":
			rank = 2
		case e.Type == "primary" || o.typ == "primary":
			rank = 1
		default:
			if nonUnique == "" {
				nonUnique = e.Name
			}
		}
		if rank > bestRank {
			bestRank = rank
			bestL, bestR = e.Field, o.field
		}
	}
	if bestRank > 0 {
		return bestL, bestR, nil
	}
	if nonUnique != "" {
		return "", "", fmt.Errorf("shared entity %q is not a primary key", nonUnique)
	}
	return "", "", fmt.Errorf("no shared entity")
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

// lookupMeasure resolves a measure, or a simple, ltm, ratio, or derived metric.
// A type:ltm metric becomes an ltm aggregation of the measure it names.
// A ratio is numerator / denominator. A derived metric is its expression.
// cumulative stays unresolved.
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
		if met.Name != local {
			continue
		}
		switch met.Type {
		case "ratio":
			return &parser.Measure{Name: met.Name, Title: met.Title, Expression: met.Numerator + " / " + met.Denominator}, model
		case "derived":
			return &parser.Measure{Name: met.Name, Title: met.Title, Expression: met.Expression}, model
		}
		if met.Measure == "" || (met.Type != "ltm" && met.Type != "simple") {
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
