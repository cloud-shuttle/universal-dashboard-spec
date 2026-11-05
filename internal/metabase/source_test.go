package metabase

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func TestModelSourceJoinColumnsTitleAndLTMMetric(t *testing.T) {
	reg := parser.NewRegistry()
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "orders", Title: "Orders"},
		Spec: parser.SemanticModelSpec{
			Source:   &parser.Source{Database: "Analytics", Schema: "marts", Table: "fct_orders"},
			Entities: []parser.Entity{{Name: "customer", Type: "foreign", Field: "customer_id"}},
			Measures: []parser.Measure{{Name: "revenue", Aggregation: "sum", Field: "amount"}},
			Dimensions: []parser.Dimension{
				{Name: "ordered_at", Field: "ordered_at", Type: "time", TemporalGrain: "month"},
			},
			Metrics: []parser.Metric{{Name: "revenue_ltm", Type: "ltm", Measure: "revenue"}},
		},
	})
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "customers", Title: "Customers"},
		Spec: parser.SemanticModelSpec{
			Source:     &parser.Source{Database: "Analytics", Schema: "marts", Table: "dim_customers"},
			Entities:   []parser.Entity{{Name: "customer", Type: "primary", Field: "id"}},
			Measures:   []parser.Measure{{Name: "customers", Aggregation: "count", Field: "id"}},
			Dimensions: []parser.Dimension{{Name: "region", Field: "region_name", Type: "categorical"}},
		},
	})
	d := &parser.Dashboard{
		Metadata: parser.Metadata{Name: "sales", Title: "Sales"},
		Spec: parser.Spec{
			Parameters: []parser.Parameter{{Name: "view", Type: "string", Default: "Monthly"}},
			Filters:    []parser.Filter{{Name: "region", Title: "Region", Dimension: "customers.region"}},
			Widgets: []parser.Widget{
				{
					ID: "rev", Type: "line", Title: "Revenue",
					TitleTemplate: "{{view}} Revenue",
					Measures:      []string{"orders.revenue"},
					Dimensions:    []string{"orders.ordered_at", "customers.region"},
					Filters:       []string{"region"},
					Layout:        parser.Layout{W: 16, H: 8},
				},
				{
					ID: "ltm", Type: "line", Title: "LTM",
					Measures:   []string{"orders.revenue_ltm"},
					Dimensions: []string{"orders.ordered_at"},
					Layout:     parser.Layout{W: 16, H: 8},
				},
			},
		},
	}
	files, err := Emit(d, EmitOptions{}, reg)
	if err != nil {
		t.Fatal(err)
	}
	var rev, ltm, dash string
	for name, body := range files {
		s := string(body)
		switch {
		case strings.Contains(name, "monthly_revenue"):
			rev = s
		case strings.Contains(name, "ltm"):
			ltm = s
		case strings.Contains(name, "dashboards"):
			dash = s
		}
	}
	if rev == "" || ltm == "" || dash == "" {
		t.Fatalf("missing files: %v", keys(files))
	}
	for _, want := range []string{"fct_orders", "dim_customers", "join-alias", "region_name", "ordered_at", `"="`} {
		if !strings.Contains(rev, want) {
			t.Fatalf("revenue card missing %q\n%s", want, rev)
		}
	}
	if strings.Contains(rev, "orders.ordered_at") || strings.Contains(rev, "Sample Database") || strings.Contains(rev, "ORDERS") {
		t.Fatalf("revenue card kept the default table or a qualified dimension\n%s", rev)
	}
	if !strings.Contains(rev, "Monthly Revenue") {
		t.Fatalf("title template was not substituted\n%s", rev)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(rev), &parsed); err != nil {
		t.Fatal(err)
	}
	vs := parsed["visualization_settings"].(map[string]any)
	dims := vs["graph.dimensions"].([]any)
	if len(dims) != 2 || dims[0] != "ordered_at" || dims[1] != "region_name" {
		t.Fatalf("graph.dimensions = %#v", dims)
	}
	metrics := vs["graph.metrics"].([]any)
	if len(metrics) != 1 || metrics[0] != "revenue" {
		t.Fatalf("graph.metrics = %#v", metrics)
	}
	if strings.Contains(dash, "template-tag") || !strings.Contains(dash, "region_name") {
		t.Fatalf("filter mapping should target the field\n%s", dash)
	}
	if !strings.Contains(ltm, "offset") || !strings.Contains(ltm, "fct_orders") {
		t.Fatalf("ltm metric was not a trailing window\n%s", ltm)
	}
	for _, card := range []string{rev, ltm} {
		for _, line := range strings.Split(card, "\n") {
			trim := strings.TrimSpace(line)
			if trim == "=" || trim == "-" || strings.HasSuffix(trim, " =") || trim == "- -" {
				t.Fatalf("bare operator %q\n%s", trim, card)
			}
		}
	}
}

func TestJoinRequiresSharedEntity(t *testing.T) {
	reg := parser.NewRegistry()
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "orders", Title: "Orders"},
		Spec: parser.SemanticModelSpec{
			Source:   &parser.Source{Database: "Analytics", Schema: "marts", Table: "fct_orders"},
			Measures: []parser.Measure{{Name: "revenue", Aggregation: "sum", Field: "amount"}},
		},
	})
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "customers", Title: "Customers"},
		Spec: parser.SemanticModelSpec{
			Source:     &parser.Source{Database: "Analytics", Schema: "marts", Table: "dim_customers"},
			Dimensions: []parser.Dimension{{Name: "region", Field: "region_name", Type: "categorical"}},
		},
	})
	d := &parser.Dashboard{
		Metadata: parser.Metadata{Name: "sales", Title: "Sales"},
		Spec: parser.Spec{
			Widgets: []parser.Widget{{
				ID: "rev", Type: "line", Title: "Revenue",
				Measures:   []string{"orders.revenue"},
				Dimensions: []string{"customers.region"},
				Layout:     parser.Layout{W: 8, H: 4},
			}},
		},
	}
	_, err := Emit(d, EmitOptions{}, reg)
	if err == nil || !strings.Contains(err.Error(), "no shared entity") {
		t.Fatal(err)
	}
}
