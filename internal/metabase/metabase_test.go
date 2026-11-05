package metabase

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func TestEmitWorkbookFeatures(t *testing.T) {
	d := &parser.Dashboard{
		Metadata: parser.Metadata{Name: "workbook", Title: "Workbook"},
		Spec: parser.Spec{
			Measures: []parser.Measure{
				{Name: "mrr", Aggregation: "sum", Field: "mrr"},
				{Name: "prev_mrr", Aggregation: "sum", Field: "prev_mrr"},
				{
					Name: "lost_mrr", Aggregation: "sum", Field: "mrr",
					Where: []parser.Predicate{{Dimension: "customer_status", Operator: "eq", Value: "Lost Customer"}},
				},
				{Name: "mrr_change", Expression: "mrr - prev_mrr"},
				{
					Name: "lost_change", Expression: "mrr - prev_mrr",
					Where: []parser.Predicate{{Dimension: "customer_status", Operator: "eq", Value: "Lost Customer"}},
				},
				{Name: "revenue_ltm", Aggregation: "ltm", Field: "amount"},
				{Name: "avg_mrr", Aggregation: "avg", Field: "mrr", Where: []parser.Predicate{{Dimension: "year", Operator: "eq", Value: "2024"}}},
			},
			Dimensions: []parser.Dimension{
				{Name: "month", Field: "month_start", Type: "date", TemporalGrain: "month"},
			},
			Parameters: []parser.Parameter{{Name: "year", Type: "string", Default: "2024"}},
			Tabs:       []parser.Tab{{Name: "year", Title: "Year", TitleTemplate: "{{year}} Performance"}},
			Filters: []parser.Filter{{
				Name: "segment", Title: "Segment", Control: "multi_select", Searchable: true,
				Bindings: []string{"orders.segment", "customers.segment"},
			}},
			Widgets: []parser.Widget{
				{
					ID: "mrr_quarter", Type: "combo", Title: "MRR by Quarter",
					Measures: []string{"lost_mrr", "mrr_change"}, Dimensions: []string{"month"},
					TimeGrain: "quarter", Filters: []string{"segment"},
					Layout: parser.Layout{W: 16, H: 8},
				},
				{
					ID: "mrr_2024", Type: "number", Title: "2024 MRR",
					Measures:     []string{"mrr"},
					FixedFilters: []parser.Predicate{{Dimension: "year", Operator: "eq", Value: "2024"}},
					Layout:       parser.Layout{X: 16, W: 8, H: 4},
				},
				{
					ID: "avg_only", Type: "number", Title: "Avg",
					Measures: []string{"avg_mrr"},
					Layout:   parser.Layout{Y: 4, W: 8, H: 4},
				},
				{
					ID: "bridge", Type: "pivot", Title: "MRR Bridge",
					Measures: []string{"mrr", "lost_mrr"},
					Rows: []parser.PivotField{
						{Dimension: "segment"},
						{Measure: "mrr"},
						{Measure: "lost_mrr"},
					},
					Columns: []parser.PivotField{{Dimension: "movement"}, {Dimension: "product"}},
					Options: map[string]any{
						"boldTotals": true,
						"negatives":  "brackets",
						"conditionalColors": []any{
							map[string]any{"measure": "lost_mrr", "when": "negative", "color": "#e8590c"},
						},
					},
					Layout: parser.Layout{Y: 8, W: 16, H: 8},
				},
				{
					ID: "lost_change", Type: "number", Title: "Lost Change",
					Measures: []string{"lost_change"},
					Layout:   parser.Layout{Y: 12, W: 8, H: 4},
				},
				{
					ID: "revenue_ltm", Type: "line", Title: "LTM Revenue",
					Measures: []string{"revenue_ltm"}, Dimensions: []string{"month"},
					Layout: parser.Layout{Y: 16, W: 16, H: 8},
				},
				{
					ID: "note", Type: "text", Title: "Notes", Content: "Lost customers only.",
					Layout: parser.Layout{X: 16, Y: 4, W: 8, H: 4},
				},
			},
		},
	}

	files, err := Emit(d, EmitOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var cards, dash string
	for name, body := range files {
		if strings.Contains(name, "/cards/") && strings.Contains(name, "mrr_by_quarter") {
			cards = string(body)
		}
		if strings.Contains(name, "/dashboards/") {
			dash = string(body)
		}
	}
	if cards == "" || dash == "" {
		t.Fatalf("missing outputs: %v", keys(files))
	}
	for _, want := range []string{"sum-where", "Lost Customer", "temporal-unit: quarter", "prev_mrr"} {
		if !strings.Contains(cards, want) {
			t.Fatalf("quarter card missing %q\n%s", want, cards)
		}
	}

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(cards), &parsed); err != nil {
		t.Fatal(err)
	}

	for name, body := range files {
		s := string(body)
		switch {
		case strings.Contains(name, "2024_mrr"):
			if !strings.Contains(s, "year") || !strings.Contains(s, "\"2024\"") && !strings.Contains(s, "2024") {
				t.Fatalf("fixed filter missing:\n%s", s)
			}
		case strings.Contains(name, "avg.yaml"):
			if !strings.Contains(s, "filter:") {
				t.Fatal("avg where should fall back to a query filter")
			}
		case strings.Contains(name, "mrr_bridge"):
			if !strings.Contains(s, "display: pivot") || !strings.Contains(s, "segment") {
				t.Fatalf("pivot card missing display or breakout:\n%s", s)
			}
			if !strings.Contains(s, "pivot.show_row_totals: true") || !strings.Contains(s, "negative_in_parentheses: true") || !strings.Contains(s, "#e8590c") {
				t.Fatalf("pivot formatting missing:\n%s", s)
			}
			rows, cols, values := pivotBuckets(t, s)
			if !bucketHas(rows, "segment") || !aggAt(rows, 0) || !aggAt(rows, 1) {
				t.Fatalf("row measure moved out of rows: %#v\n%s", rows, s)
			}
			if !bucketHas(cols, "movement") || !bucketHas(cols, "product") {
				t.Fatalf("column levels missing: %#v", cols)
			}
			if aggAt(values, 0) || aggAt(values, 1) {
				t.Fatalf("row measures also landed in values: %#v", values)
			}
		case strings.Contains(name, "lost_change"):
			if strings.Count(s, "sum-where") < 2 || !strings.Contains(s, "Lost Customer") {
				t.Fatalf("expression where missing:\n%s", s)
			}
		case strings.Contains(name, "ltm_revenue"):
			if !strings.Contains(s, "offset") || !strings.Contains(s, "-11") {
				t.Fatalf("ltm window missing:\n%s", s)
			}
		case strings.Contains(name, "notes"):
			t.Fatal("text widget should not become a question card")
		}
	}
	if !strings.Contains(dash, "searchable: true") || !strings.Contains(dash, "orders.segment") {
		t.Fatalf("dashboard filter missing searchable bindings:\n%s", dash)
	}
	if !strings.Contains(dash, "virtual_card") || !strings.Contains(dash, "Lost customers only.") {
		t.Fatalf("text card missing:\n%s", dash)
	}
	if !strings.Contains(dash, "2024 Performance") || !strings.Contains(dash, "title_template") {
		t.Fatalf("tab template missing:\n%s", dash)
	}
	if !strings.Contains(dash, "customers.segment") {
		t.Fatal("expected the customers binding to be selected for no matching model, or listed on the parameter")
	}
}

func pivotBuckets(t *testing.T, card string) (rows, cols, values []any) {
	t.Helper()
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(card), &parsed); err != nil {
		t.Fatal(err)
	}
	vs, _ := parsed["visualization_settings"].(map[string]any)
	split, _ := vs["pivot_table.column_split"].(map[string]any)
	rows, _ = split["rows"].([]any)
	cols, _ = split["columns"].([]any)
	values, _ = split["values"].([]any)
	return rows, cols, values
}

func bucketHas(items []any, name string) bool {
	for _, item := range items {
		if item == name {
			return true
		}
	}
	return false
}

func aggAt(items []any, index int) bool {
	for _, item := range items {
		pair, ok := item.([]any)
		if !ok || len(pair) != 2 || pair[0] != "aggregation" {
			continue
		}
		switch n := pair[1].(type) {
		case int:
			if n == index {
				return true
			}
		case int64:
			if int(n) == index {
				return true
			}
		case uint64:
			if int(n) == index {
				return true
			}
		}
	}
	return false
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
