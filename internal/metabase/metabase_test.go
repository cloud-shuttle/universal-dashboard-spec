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
				{Name: "avg_mrr", Aggregation: "avg", Field: "mrr", Where: []parser.Predicate{{Dimension: "year", Operator: "eq", Value: "2024"}}},
			},
			Dimensions: []parser.Dimension{
				{Name: "month", Field: "month_start", Type: "date", TemporalGrain: "month"},
			},
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
					Rows: []parser.PivotField{
						{Dimension: "segment"},
						{Measure: "mrr"},
						{Measure: "lost_mrr"},
					},
					Columns: []parser.PivotField{{Dimension: "product"}},
					Layout:  parser.Layout{Y: 8, W: 16, H: 8},
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
			if !strings.Contains(s, "pivot.row_order") || !strings.Contains(s, "segment") || !strings.Contains(s, "lost_mrr") {
				t.Fatalf("pivot order missing:\n%s", s)
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
	if !strings.Contains(dash, "customers.segment") {
		t.Fatal("expected the customers binding to be selected for no matching model, or listed on the parameter")
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
