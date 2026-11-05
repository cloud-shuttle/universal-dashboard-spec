package metabase

import (
	"strconv"
	"strings"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func buildMBQL(w parser.Widget, d *parser.Dashboard, db, schema, table string, reg *parser.Registry) map[string]any {
	agg := []any{}
	var whereFallback []parser.Predicate
	for _, mName := range widgetMeasures(w) {
		m := findMeasure(d, mName, reg)
		if m == nil {
			continue
		}
		item, fallback := measureAgg(m, d, db, schema, table, reg, map[string]bool{}, cardGrain(w, d, reg), nil)
		agg = append(agg, item)
		whereFallback = append(whereFallback, fallback...)
	}

	breakout := []any{}
	seenDim := map[string]bool{}
	addDim := func(dimName string, allowMissing bool) {
		if dimName == "" || seenDim[dimName] {
			return
		}
		dim := findDimension(d, dimName, reg)
		if dim == nil && !allowMissing {
			return
		}
		seenDim[dimName] = true
		field := dimName
		var grain string
		if dim != nil {
			if dim.Field != "" {
				field = dim.Field
			}
			grain = dim.TemporalGrain
			if w.TimeGrain != "" && temporalDim(dim, len(w.Dimensions)) {
				grain = w.TimeGrain
			}
		}
		ref := fieldRef(db, schema, table, field)
		if grain != "" {
			ref = []any{"field", ref[1], map[string]any{"temporal-unit": grain}}
		}
		breakout = append(breakout, ref)
	}
	for _, dimName := range w.Dimensions {
		addDim(dimName, false)
	}
	for _, item := range w.Rows {
		addDim(item.Dimension, true)
	}
	for _, item := range w.Columns {
		addDim(item.Dimension, true)
	}

	query := map[string]any{
		"source-table": []any{db, schema, table},
		"aggregation":  agg,
		"breakout":     breakout,
	}

	var filters []any
	// ponytail: avg/min/max have no MBQL *-where; apply the predicate on the query only for a one-measure card.
	if len(widgetMeasures(w)) == 1 && len(whereFallback) > 0 {
		filters = append(filters, predicatesMBQL(whereFallback, d, db, schema, table, reg))
	}
	if len(w.FixedFilters) > 0 {
		filters = append(filters, predicatesMBQL(w.FixedFilters, d, db, schema, table, reg))
	}
	if len(filters) == 1 {
		query["filter"] = filters[0]
	} else if len(filters) > 1 {
		query["filter"] = append([]any{"and"}, filters...)
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

func temporalDim(dim *parser.Dimension, dimCount int) bool {
	switch dim.Type {
	case "date", "time":
		return true
	}
	if dim.TemporalGrain != "" || len(dim.Grains) > 0 {
		return true
	}
	return dimCount == 1
}

// measureAgg returns the MBQL aggregation and, for aggregations with no
// where-variant, the predicates the caller may apply at query level.
// inherit is the where from an expression that uses this measure. It is not
// written back onto the measure, so the same closing measure can stay
// unfiltered on one card and be filtered inside lost_diff.
func measureAgg(m *parser.Measure, d *parser.Dashboard, db, schema, table string, reg *parser.Registry, visiting map[string]bool, grain string, inherit []parser.Predicate) (any, []parser.Predicate) {
	wheres := append(append([]parser.Predicate{}, inherit...), m.Where...)
	if m.Expression != "" {
		if visiting[m.Name] {
			return []any{"sum", fieldRef(db, schema, table, m.Name)}, nil
		}
		visiting[m.Name] = true
		ex, err := parser.ParseExpression(m.Expression)
		if err != nil {
			delete(visiting, m.Name)
			return []any{"sum", fieldRef(db, schema, table, m.Name)}, nil
		}
		item := compileExpr(ex, d, db, schema, table, reg, visiting, grain, wheres)
		delete(visiting, m.Name)
		return item, nil
	}
	aggName := m.Aggregation
	if aggName == "" {
		aggName = "sum"
	}
	fieldName := m.Field
	if fieldName == "" {
		fieldName = m.Name
	}
	field := fieldRef(db, schema, table, fieldName)
	if aggName == "ltm" {
		return ltmAgg(m.Name, field, wheres, grain, d, db, schema, table, reg), nil
	}
	if len(wheres) == 0 {
		return []any{aggName, named(m.Name), field}, nil
	}
	clause := predicatesMBQL(wheres, d, db, schema, table, reg)
	switch aggName {
	case "sum":
		return []any{"sum-where", named(m.Name), field, clause}, nil
	case "count":
		return []any{"count-where", named(m.Name), field, clause}, nil
	case "count_distinct":
		return []any{"distinct-where", named(m.Name), field, clause}, nil
	default:
		// ponytail: avg/min/max have no MBQL *-where form.
		return []any{aggName, named(m.Name), field}, wheres
	}
}

func named(name string) map[string]any {
	if name == "" {
		return map[string]any{}
	}
	return map[string]any{"name": name}
}

func compileExpr(e *parser.Expr, d *parser.Dashboard, db, schema, table string, reg *parser.Registry, visiting map[string]bool, grain string, inherit []parser.Predicate) any {
	switch {
	case e.Name != "":
		if m := findMeasure(d, e.Name, reg); m != nil {
			item, _ := measureAgg(m, d, db, schema, table, reg, visiting, grain, inherit)
			return item
		}
		field := fieldRef(db, schema, table, e.Name)
		if len(inherit) == 0 {
			return []any{"sum", named(e.Name), field}
		}
		return []any{"sum-where", named(e.Name), field, predicatesMBQL(inherit, d, db, schema, table, reg)}
	case e.Num != "":
		if i, err := strconv.Atoi(e.Num); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(e.Num, 64); err == nil {
			return f
		}
		return e.Num
	case e.Op == "neg":
		return []any{"-", compileExpr(e.Left, d, db, schema, table, reg, visiting, grain, inherit)}
	default:
		return []any{
			e.Op,
			compileExpr(e.Left, d, db, schema, table, reg, visiting, grain, inherit),
			compileExpr(e.Right, d, db, schema, table, reg, visiting, grain, inherit),
		}
	}
}

// ltmAgg is a trailing window of the breakout grain that adds up to twelve
// months: 12 months, 4 quarters, 52 weeks, or 365 days. A yearly breakout is
// already one bucket, so it stays a single sum.
func ltmAgg(name string, field any, wheres []parser.Predicate, grain string, d *parser.Dashboard, db, schema, table string, reg *parser.Registry) any {
	base := []any{"sum", field}
	if len(wheres) > 0 {
		base = []any{"sum-where", field, predicatesMBQL(wheres, d, db, schema, table, reg)}
	}
	periods := ltmPeriods(grain)
	if periods == 1 {
		return append([]any{base[0], named(name)}, base[1:]...)
	}
	terms := []any{"+", named(name), base}
	for i := 1; i < periods; i++ {
		terms = append(terms, []any{"offset", map[string]any{}, base, -i})
	}
	return terms
}

func ltmPeriods(grain string) int {
	switch grain {
	case "day":
		return 365
	case "week":
		return 52
	case "quarter":
		return 4
	case "year":
		return 1
	default:
		return 12
	}
}

func cardGrain(w parser.Widget, d *parser.Dashboard, reg *parser.Registry) string {
	if w.TimeGrain != "" {
		return w.TimeGrain
	}
	for _, dimName := range w.Dimensions {
		dim := findDimension(d, dimName, reg)
		if dim != nil && dim.TemporalGrain != "" {
			return dim.TemporalGrain
		}
	}
	return "month"
}

func predicatesMBQL(ps []parser.Predicate, d *parser.Dashboard, db, schema, table string, reg *parser.Registry) any {
	parts := make([]any, 0, len(ps))
	for _, p := range ps {
		ref := fieldRef(db, schema, table, predicateField(p.Dimension, d, reg))
		var clause any
		switch p.Operator {
		case "neq":
			clause = []any{"!=", ref, p.Value}
		case "gt":
			clause = []any{">", ref, p.Value}
		case "gte":
			clause = []any{">=", ref, p.Value}
		case "lt":
			clause = []any{"<", ref, p.Value}
		case "lte":
			clause = []any{"<=", ref, p.Value}
		case "in", "not_in":
			args := []any{"in", ref}
			switch v := p.Value.(type) {
			case []any:
				for _, item := range v {
					args = append(args, item)
				}
			default:
				args = append(args, p.Value)
			}
			if p.Operator == "not_in" {
				clause = []any{"not", args}
			} else {
				clause = args
			}
		default:
			clause = []any{"=", ref, p.Value}
		}
		parts = append(parts, clause)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return append([]any{"and"}, parts...)
}

func predicateField(name string, d *parser.Dashboard, reg *parser.Registry) string {
	if dim := findDimension(d, name, reg); dim != nil && dim.Field != "" {
		return dim.Field
	}
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}

func fieldRef(db, schema, table, field string) []any {
	return []any{"field", []any{db, schema, table, field}, nil}
}
