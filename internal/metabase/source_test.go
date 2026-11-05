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

func TestFilterBindingDoesNotJoin(t *testing.T) {
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
		},
	})
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "customers", Title: "Customers"},
		Spec: parser.SemanticModelSpec{
			Source:     &parser.Source{Database: "Analytics", Schema: "marts", Table: "dim_customers"},
			Entities:   []parser.Entity{{Name: "customer", Type: "primary", Field: "id"}},
			Dimensions: []parser.Dimension{{Name: "region", Field: "region_name", Type: "categorical"}},
		},
	})
	d := &parser.Dashboard{
		Metadata: parser.Metadata{Name: "sales", Title: "Sales"},
		Spec: parser.Spec{
			Filters: []parser.Filter{{Name: "region", Title: "Region", Bindings: []string{"customers.region"}}},
			Widgets: []parser.Widget{{
				ID: "rev", Type: "line", Title: "Revenue",
				Measures:   []string{"orders.revenue"},
				Dimensions: []string{"orders.ordered_at"},
				Filters:    []string{"region"},
				Layout:     parser.Layout{W: 8, H: 4},
			}},
		},
	}
	files, err := Emit(d, EmitOptions{}, reg)
	if err != nil {
		t.Fatal(err)
	}
	var card, dash string
	for name, body := range files {
		s := string(body)
		if strings.Contains(name, "dashboards") {
			dash = s
		} else {
			card = s
		}
	}
	if strings.Contains(card, "left-join") || strings.Contains(card, "join-alias") || strings.Contains(card, "dim_customers") {
		t.Fatalf("filter binding added a join\n%s", card)
	}
	if strings.Contains(dash, "join-alias") || strings.Contains(dash, "region_name") {
		t.Fatalf("unmapped filter still targeted the other model\n%s", dash)
	}
}

func TestJoinRefusesForeignForeign(t *testing.T) {
	reg := parser.NewRegistry()
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "retention", Title: "Retention"},
		Spec: parser.SemanticModelSpec{
			Source:   &parser.Source{Database: "Analytics", Schema: "marts", Table: "fct_retention"},
			Entities: []parser.Entity{{Name: "customer", Type: "foreign", Field: "customer"}},
			Measures: []parser.Measure{{Name: "retained", Aggregation: "sum", Field: "retained"}},
		},
	})
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "revenue", Title: "Revenue"},
		Spec: parser.SemanticModelSpec{
			Source:     &parser.Source{Database: "Analytics", Schema: "marts", Table: "fct_revenue"},
			Entities:   []parser.Entity{{Name: "customer", Type: "foreign", Field: "customer"}},
			Dimensions: []parser.Dimension{{Name: "product", Field: "product", Type: "categorical"}},
		},
	})
	d := &parser.Dashboard{
		Metadata: parser.Metadata{Name: "sales", Title: "Sales"},
		Spec: parser.Spec{
			Widgets: []parser.Widget{{
				ID: "ret", Type: "line", Title: "Retained",
				Measures:   []string{"retention.retained"},
				Dimensions: []string{"revenue.product"},
				Layout:     parser.Layout{W: 8, H: 4},
			}},
		},
	}
	_, err := Emit(d, EmitOptions{}, reg)
	if err == nil || !strings.Contains(err.Error(), "not a primary key") {
		t.Fatal(err)
	}
}

func TestRatioAndDerivedMetrics(t *testing.T) {
	reg := parser.NewRegistry()
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "orders", Title: "Orders"},
		Spec: parser.SemanticModelSpec{
			Source: &parser.Source{Database: "Analytics", Schema: "marts", Table: "fct_orders"},
			Measures: []parser.Measure{
				{Name: "revenue", Aggregation: "sum", Field: "amount"},
				{Name: "refunds", Aggregation: "sum", Field: "refund_amount"},
				{Name: "order_count", Aggregation: "count", Field: "id"},
			},
			Metrics: []parser.Metric{
				{Name: "aov", Type: "ratio", Numerator: "revenue", Denominator: "order_count"},
				{Name: "net", Type: "derived", Expression: "revenue - refunds"},
			},
		},
	})
	d := &parser.Dashboard{
		Metadata: parser.Metadata{Name: "sales", Title: "Sales"},
		Spec: parser.Spec{
			Widgets: []parser.Widget{
				{
					ID: "aov", Type: "number", Title: "AOV",
					Measures: []string{"orders.aov"},
					Layout:   parser.Layout{W: 6, H: 4},
				},
				{
					ID: "net", Type: "number", Title: "Net",
					Measures: []string{"orders.net"},
					Layout:   parser.Layout{W: 6, H: 4},
				},
			},
		},
	}
	files, err := Emit(d, EmitOptions{}, reg)
	if err != nil {
		t.Fatal(err)
	}
	var aov, net string
	for name, body := range files {
		s := string(body)
		switch {
		case strings.Contains(name, "aov"):
			aov = s
		case strings.Contains(name, "net"):
			net = s
		}
	}
	if aov == "" || net == "" {
		t.Fatalf("missing files: %v", keys(files))
	}
	if strings.Contains(aov, "aggregation: []") || !strings.Contains(aov, `"/"`) || !strings.Contains(aov, "amount") || !strings.Contains(aov, "id") {
		t.Fatalf("ratio metric was dropped\n%s", aov)
	}
	if strings.Contains(net, "aggregation: []") || !strings.Contains(net, `"-"`) || !strings.Contains(net, "amount") || !strings.Contains(net, "refund_amount") {
		t.Fatalf("derived metric was dropped\n%s", net)
	}
}

func TestMissingDatabaseWarnsAndFlagFillsJoins(t *testing.T) {
	reg := parser.NewRegistry()
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "orders", Title: "Orders"},
		Spec: parser.SemanticModelSpec{
			Source:   &parser.Source{Schema: "marts", Table: "fct_orders"},
			Entities: []parser.Entity{{Name: "customer", Type: "foreign", Field: "customer_id"}},
			Measures: []parser.Measure{{Name: "revenue", Aggregation: "sum", Field: "amount"}},
		},
	})
	reg.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "customers", Title: "Customers"},
		Spec: parser.SemanticModelSpec{
			Source:     &parser.Source{Schema: "marts", Table: "dim_customers"},
			Entities:   []parser.Entity{{Name: "customer", Type: "primary", Field: "id"}},
			Dimensions: []parser.Dimension{{Name: "region", Field: "region_name", Type: "categorical"}},
		},
	})
	w := parser.Widget{
		ID: "rev", Type: "line", Title: "Revenue",
		Measures:   []string{"orders.revenue"},
		Dimensions: []string{"customers.region"},
		Layout:     parser.Layout{W: 8, H: 4},
	}
	d := &parser.Dashboard{
		Metadata: parser.Metadata{Name: "sales", Title: "Sales"},
		Spec:     parser.Spec{Widgets: []parser.Widget{w}},
	}
	src, err := resolveCardSource(w, d, EmitOptions{}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if src.db != "Sample Database" || len(src.joins) != 1 || src.joins[0].db != "Sample Database" {
		t.Fatalf("default database = %s joins %#v", src.db, src.joins)
	}
	warns := strings.Join(src.warnings, "\n")
	if len(src.warnings) != 2 || !strings.Contains(warns, `model "orders" has no database`) || !strings.Contains(warns, `model "customers" has no database`) || !strings.Contains(warns, "Sample Database") {
		t.Fatalf("warnings = %#v", src.warnings)
	}
	filled, err := resolveCardSource(w, d, EmitOptions{Database: "Other"}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if filled.db != "Other" || len(filled.joins) != 1 || filled.joins[0].db != "Other" {
		t.Fatalf("flag database = %s joins %#v", filled.db, filled.joins)
	}
	if filled.joins[0].table != "dim_customers" || filled.table != "fct_orders" {
		t.Fatalf("flag rewrote tables: %s %s", filled.table, filled.joins[0].table)
	}
	if len(filled.warnings) != 0 {
		t.Fatalf("database flag still warned: %#v", filled.warnings)
	}

	bare := parser.NewRegistry()
	bare.Add(&parser.SemanticModel{
		Metadata: parser.Metadata{Name: "recon", Title: "Recon"},
		Spec: parser.SemanticModelSpec{
			Measures: []parser.Measure{{Name: "rows", Aggregation: "count", Field: "id"}},
		},
	})
	rw := parser.Widget{
		ID: "recon", Type: "number", Title: "Recon",
		Measures: []string{"recon.rows"},
		Layout:   parser.Layout{W: 6, H: 4},
	}
	rs, err := resolveCardSource(rw, d, EmitOptions{}, bare)
	if err != nil {
		t.Fatal(err)
	}
	if rs.table != "ORDERS" || len(rs.warnings) != 1 || !strings.Contains(rs.warnings[0], "has no source") || !strings.Contains(rs.warnings[0], "ORDERS") {
		t.Fatalf("sourceless model = %s.%s.%s warnings %#v", rs.db, rs.schema, rs.table, rs.warnings)
	}
}
