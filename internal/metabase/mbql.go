package metabase

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func buildMBQL(w parser.Widget, d *parser.Dashboard, src cardSource, reg *parser.Registry) (map[string]any, error) {
	agg := []any{}
	var whereFallback []parser.Predicate
	var fallbackModel string
	for _, mName := range widgetMeasures(w) {
		m, model := lookupMeasure(d, mName, reg)
		if m == nil {
			fmt.Fprintf(os.Stderr, "warning: widget %q: measure %q not found, skipped\n", w.ID, mName)
			continue
		}
		if m.PriorPeriod != nil && !breakoutHas(w, m.PriorPeriod.Dimension) {
			fmt.Fprintf(os.Stderr, "warning: widget %q: measure %q priorPeriod dimension %q is not a breakout\n", w.ID, m.Name, m.PriorPeriod.Dimension)
		}
		item, fallback := measureAgg(m, d, src, reg, map[string]bool{}, cardGrain(w, d, reg), nil, model)
		agg = append(agg, item)
		if len(fallback) > 0 && fallbackModel == "" {
			fallbackModel = model
		}
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
		model, field, _ := splitRef(dimName)
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
		var extra map[string]any
		if grain != "" {
			extra = map[string]any{"temporal-unit": grain}
		}
		breakout = append(breakout, src.field(model, field, extra))
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
		"aggregation": agg,
		"breakout":    breakout,
	}
	if src.function != "" {
		query["source-query"] = nativeSource(src, d)
	} else {
		query["source-table"] = []any{src.db, src.schema, src.table}
		if joins := src.joinClauses(); len(joins) > 0 {
			query["joins"] = joins
		}
	}

	var filters []any
	// ponytail: avg/min/max have no MBQL *-where; apply the predicate on the query only for a one-measure card.
	if len(widgetMeasures(w)) == 1 && len(whereFallback) > 0 {
		filters = append(filters, predicatesMBQL(whereFallback, d, src, reg, fallbackModel))
	}
	if len(w.FixedFilters) > 0 {
		filters = append(filters, predicatesMBQL(w.FixedFilters, d, src, reg, src.base))
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
		"database": src.db,
		"type":     "query",
		"query":    query,
	}, nil
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
func measureAgg(m *parser.Measure, d *parser.Dashboard, src cardSource, reg *parser.Registry, visiting map[string]bool, grain string, inherit []parser.Predicate, model string) (any, []parser.Predicate) {
	wheres := append(append([]parser.Predicate{}, inherit...), m.Where...)
	if m.Expression != "" {
		if visiting[m.Name] {
			return []any{"sum", src.field(model, m.Name, nil)}, nil
		}
		visiting[m.Name] = true
		ex, err := parser.ParseExpression(m.Expression)
		if err != nil {
			delete(visiting, m.Name)
			return []any{"sum", src.field(model, m.Name, nil)}, nil
		}
		item := nameAgg(m.Name, compileExpr(ex, d, src, reg, visiting, grain, wheres, model))
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
	field := src.field(model, fieldName, nil)
	if m.PriorPeriod != nil {
		return priorAgg(m.Name, aggName, field, wheres, m.PriorPeriod.Offset, d, src, reg, model), nil
	}
	if aggName == "ltm" {
		return ltmAgg(m.Name, field, wheres, grain, d, src, reg, model), nil
	}
	if len(wheres) == 0 {
		return []any{aggName, named(m.Name), field}, nil
	}
	clause := predicatesMBQL(wheres, d, src, reg, model)
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

// nameAgg attaches the measure name to an expression aggregation so the result
// column matches graph.metrics. Aggregations that already carry a name are left as they are.
func nameAgg(name string, item any) any {
	parts, ok := item.([]any)
	if !ok || name == "" || len(parts) == 0 {
		return item
	}
	if parts[0] == "case" {
		if opts, ok := parts[len(parts)-1].(map[string]any); ok {
			opts["name"] = name
			return parts
		}
	}
	if len(parts) > 1 {
		if _, ok := parts[1].(map[string]any); ok {
			return item
		}
	}
	out := make([]any, 0, len(parts)+1)
	out = append(out, parts[0], named(name))
	out = append(out, parts[1:]...)
	return out
}

func compileExpr(e *parser.Expr, d *parser.Dashboard, src cardSource, reg *parser.Registry, visiting map[string]bool, grain string, inherit []parser.Predicate, model string) any {
	switch {
	case e.Name != "":
		ref := e.Name
		if model != "" && !strings.Contains(e.Name, ".") {
			ref = model + "." + e.Name
		}
		m, _ := lookupMeasure(d, ref, reg)
		owner := model
		if m == nil && ref != e.Name {
			m, _ = lookupMeasure(d, e.Name, reg)
			owner = ""
		}
		if m != nil {
			if om, _, ok := splitRef(ref); ok && owner == model {
				owner = om
			}
			item, _ := measureAgg(m, d, src, reg, visiting, grain, inherit, owner)
			return item
		}
		field := src.field(model, e.Name, nil)
		if len(inherit) == 0 {
			return []any{"sum", named(e.Name), field}
		}
		return []any{"sum-where", named(e.Name), field, predicatesMBQL(inherit, d, src, reg, model)}
	case e.Num != "":
		if i, err := strconv.Atoi(e.Num); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(e.Num, 64); err == nil {
			return f
		}
		return e.Num
	case e.Op == "neg":
		return []any{"-", compileExpr(e.Left, d, src, reg, visiting, grain, inherit, model)}
	default:
		left := compileExpr(e.Left, d, src, reg, visiting, grain, inherit, model)
		right := compileExpr(e.Right, d, src, reg, visiting, grain, inherit, model)
		if e.Op == "/" {
			// A zero denominator is undefined, so the rate is null rather than 0.
			return []any{
				"case",
				[]any{[]any{[]any{"=", right, 0}, nil}},
				map[string]any{"default": []any{"/", left, right}},
			}
		}
		return []any{e.Op, left, right}
	}
}

func breakoutHas(w parser.Widget, dimension string) bool {
	for _, name := range w.Dimensions {
		if name == dimension || shortName(name) == dimension {
			return true
		}
	}
	for _, item := range append(append([]parser.PivotField{}, w.Rows...), w.Columns...) {
		if item.Dimension == dimension || shortName(item.Dimension) == dimension {
			return true
		}
	}
	return false
}

func priorAgg(name, aggName string, field any, wheres []parser.Predicate, offset int, d *parser.Dashboard, src cardSource, reg *parser.Registry, model string) any {
	base := []any{aggName, field}
	if len(wheres) > 0 {
		clause := predicatesMBQL(wheres, d, src, reg, model)
		// ponytail: avg/min/max have no *-where, so a priorPeriod of those drops the where. Add when a spec needs it.
		switch aggName {
		case "sum", "count", "count_distinct":
			base = []any{aggName + "-where", field, clause}
		}
	}
	if offset == 0 {
		return append([]any{base[0], named(name)}, base[1:]...)
	}
	return []any{"offset", named(name), base, offset}
}

// ltmAgg is a trailing window of the breakout grain that adds up to twelve
// months: 12 months, 4 quarters, 52 weeks, or 365 days. A yearly breakout is
// already one bucket, so it stays a single sum.
func ltmAgg(name string, field any, wheres []parser.Predicate, grain string, d *parser.Dashboard, src cardSource, reg *parser.Registry, model string) any {
	base := []any{"sum", field}
	if len(wheres) > 0 {
		base = []any{"sum-where", field, predicatesMBQL(wheres, d, src, reg, model)}
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

func predicatesMBQL(ps []parser.Predicate, d *parser.Dashboard, src cardSource, reg *parser.Registry, owner string) any {
	parts := make([]any, 0, len(ps))
	for _, p := range ps {
		model, local, qualified := splitRef(p.Dimension)
		if !qualified {
			model = owner
			local = p.Dimension
		}
		lookup := p.Dimension
		if model != "" && !qualified {
			lookup = model + "." + local
		}
		field := local
		if dim := findDimension(d, lookup, reg); dim != nil && dim.Field != "" {
			field = dim.Field
		} else if dim := findDimension(d, p.Dimension, reg); dim != nil && dim.Field != "" {
			field = dim.Field
		}
		ref := src.field(model, field, nil)
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
