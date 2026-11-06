package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidDashboard(t *testing.T) {
	path := writeTemp(t, validDashboard)
	d, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if d.Metadata.Name != "sales-overview" {
		t.Fatalf("name = %q", d.Metadata.Name)
	}
	if len(d.Spec.Widgets) != 1 {
		t.Fatalf("widgets = %d", len(d.Spec.Widgets))
	}
}

func TestLoadRejectsUnknownIntent(t *testing.T) {
	path := writeTemp(t, invalidIntent)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected schema error for unknown intent")
	}
}

func TestLoadParsesParametersAndFilters(t *testing.T) {
	path := writeTemp(t, withParams)
	d, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Spec.Parameters) != 1 {
		t.Fatalf("parameters = %d", len(d.Spec.Parameters))
	}
	p := d.Spec.Parameters[0]
	if p.Affects != "measure" || len(p.Values) != 2 {
		t.Fatalf("parameter = %+v", p)
	}
	if d.Spec.Filters[0].Control != "radio" {
		t.Fatalf("control = %q", d.Spec.Filters[0].Control)
	}
}

func TestLoadParsesTitleTemplateAndSeries(t *testing.T) {
	path := writeTemp(t, withSeries)
	d, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	w := d.Spec.Widgets[0]
	if w.TitleTemplate != "{{period}} ARR" {
		t.Fatalf("titleTemplate = %q", w.TitleTemplate)
	}
	if len(w.Series) != 1 || w.Series[0].Color != "#1f77b4" {
		t.Fatalf("series = %+v", w.Series)
	}
}

func TestValidateIntentMismatchIsWarning(t *testing.T) {
	path := writeTemp(t, mismatchedIntent)
	d, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	warnings, err := d.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for mismatched intent/type")
	}
}

func TestEmbeddedSchemasMatch(t *testing.T) {
	pairs := [][2]string{
		{"schema.json", "../../schema/dashboard.json"},
		{"semantic-model.json", "../../schema/semantic-model.json"},
	}
	for _, pair := range pairs {
		a, err := os.ReadFile(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Fatalf("%s differs from %s", pair[0], pair[1])
		}
	}
}

func TestFunctionArgumentsKeepYAMLOrder(t *testing.T) {
	path := writeTemp(t, `
apiVersion: uds/v1
kind: SemanticModel
metadata:
  name: rank
  title: Rank
spec:
  source:
    database: Warehouse
    function: analytics.orders_for
    arguments:
      p_as_of: { parameter: as_of_month }
      p_region: { filter: region, default: All }
      p_window: { parameter: compare, map: { year: 12, quarter: 3 } }
  dimensions:
    - { name: rank, type: number, field: rank }
  measures:
    - { name: rows, aggregation: count, field: customer }
`)
	m, err := LoadSemanticModel(path)
	if err != nil {
		t.Fatal(err)
	}
	args := m.Spec.Source.Arguments
	if m.Spec.Source.Function != "analytics.orders_for" || len(args) != 3 {
		t.Fatalf("source = %+v", m.Spec.Source)
	}
	if args[0].Name != "p_as_of" || args[1].Name != "p_region" || args[1].Default != "All" || args[2].Name != "p_window" {
		t.Fatalf("arguments = %+v", args)
	}
	if args[2].Map["year"] != 12 && args[2].Map["year"] != int64(12) && fmtNumber(args[2].Map["year"]) != "12" {
		t.Fatalf("map = %#v", args[2].Map)
	}
}

func fmtNumber(v any) string {
	return strings.TrimSpace(strings.ReplaceAll(fmt.Sprint(v), "\n", ""))
}

func TestExamplesLoad(t *testing.T) {
	dash, err := Load("../../examples/sales-overview-semantic.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(dash.Spec.Models) != 1 || dash.Spec.Models[0] != "orders" {
		t.Fatalf("models = %v", dash.Spec.Models)
	}

	model, err := LoadSemanticModel("../../examples/orders.semantic.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Validate(); err != nil {
		t.Fatal(err)
	}
	var number, lost, change bool
	for _, d := range model.Spec.Dimensions {
		if d.Name == "amount" && d.Type == "number" {
			number = true
		}
		if d.Name == "ordered_at" && len(d.Grains) != 5 {
			t.Fatalf("grains = %v", d.Grains)
		}
	}
	for _, m := range model.Spec.Measures {
		if m.Name == "lost_revenue" && len(m.Where) == 1 && m.Where[0].Value == "Lost Customer" {
			lost = true
		}
		if m.Name == "revenue_change" && m.Expression == "revenue - lost_revenue" {
			change = true
		}
	}
	if !number || !lost || !change {
		t.Fatalf("number=%v lost=%v change=%v", number, lost, change)
	}
}

func TestWorkbookExample(t *testing.T) {
	d, err := Load("../../examples/workbook.yaml")
	if err != nil {
		t.Fatal(err)
	}
	warnings, err := d.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if d.Spec.Widgets[0].TimeGrain != "quarter" {
		t.Fatalf("timeGrain = %q", d.Spec.Widgets[0].TimeGrain)
	}
	if d.Spec.Tabs[0].TitleTemplate != "{{year}} Performance" {
		t.Fatalf("tab template = %q", d.Spec.Tabs[0].TitleTemplate)
	}
	if !d.Spec.Filters[0].Searchable || len(d.Spec.Filters[0].Bindings) != 2 {
		t.Fatalf("filter = %+v", d.Spec.Filters[0])
	}
	bridge := d.Spec.Widgets[2]
	if bridge.Rows[0].Dimension != "segment" || bridge.Rows[1].Measure != "mrr" {
		t.Fatalf("rows = %+v", bridge.Rows)
	}
}

func TestValidateExpressionRules(t *testing.T) {
	if _, err := ParseExpression("mrr - prev_mrr"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseExpression("(a + b) * 2"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseExpression("sum(mrr)"); err == nil {
		t.Fatal("expected function call to fail")
	}

	both := &Dashboard{Spec: Spec{Measures: []Measure{{
		Name: "m", Aggregation: "sum", Expression: "mrr - prev_mrr",
	}}}}
	if _, err := both.Validate(); err == nil {
		t.Fatal("expected aggregation/expression conflict")
	}

	bad := &Dashboard{Spec: Spec{Measures: []Measure{{
		Name: "m", Expression: "mrr; drop table",
	}}}}
	if _, err := bad.Validate(); err == nil {
		t.Fatal("expected bad expression")
	}
}

func TestComboIntentWarnings(t *testing.T) {
	ok := &Dashboard{Spec: Spec{
		Measures: []Measure{{Name: "m", Aggregation: "sum"}},
		Widgets:  []Widget{{ID: "c", Type: "combo", Intent: "time_series", Title: "C"}},
	}}
	warnings, err := ok.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}

	rank := &Dashboard{Spec: Spec{
		Measures: []Measure{{Name: "m", Aggregation: "sum"}},
		Widgets:  []Widget{{ID: "c", Type: "combo", Intent: "ranking", Title: "C"}},
	}}
	warnings, err = rank.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestSourceOrDefault(t *testing.T) {
	d := &Dashboard{Spec: Spec{Source: &Source{Database: "X", Table: "T"}}}
	db, schema, table := d.SourceOrDefault("defDB", "defSchema", "defTable")
	if db != "X" || schema != "defSchema" || table != "T" {
		t.Fatalf("got %s/%s/%s", db, schema, table)
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "dash.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validDashboard = `
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: sales-overview
  title: Sales Overview
spec:
  measures:
    - name: revenue
      aggregation: sum
      field: amount
  dimensions:
    - name: closed_date
      type: date
  widgets:
    - id: rev
      type: line
      intent: time_series
      title: Revenue
      measures: [revenue]
      dimensions: [closed_date]
      layout: { x: 0, y: 0, w: 12, h: 6 }
`

const invalidIntent = `
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: bad
  title: Bad
spec:
  measures:
    - name: m
      aggregation: sum
  dimensions:
    - name: d
  widgets:
    - id: w
      type: line
      intent: not_a_real_intent
      title: W
      layout: { x: 0, y: 0, w: 12, h: 6 }
`

const withParams = `
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: p
  title: P
spec:
  parameters:
    - name: period
      type: string
      values: [Monthly, LTM]
      affects: measure
  measures:
    - name: m
      aggregation: sum
  dimensions:
    - name: d
  filters:
    - name: recurring
      dimension: d
      control: radio
  widgets:
    - id: w
      type: number
      title: W
      layout: { x: 0, y: 0, w: 6, h: 4 }
`

const withSeries = `
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: s
  title: S
spec:
  measures:
    - name: m
      aggregation: sum
  dimensions:
    - name: d
  widgets:
    - id: w
      type: line
      title: Monthly ARR
      titleTemplate: "{{period}} ARR"
      measures: [m]
      dimensions: [d]
      series:
        - name: revenue
          color: "#1f77b4"
          tooltipFields: [deal_count]
      layout: { x: 0, y: 0, w: 12, h: 6 }
`

const mismatchedIntent = `
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: m
  title: M
spec:
  measures:
    - name: m
      aggregation: sum
  dimensions:
    - name: d
  widgets:
    - id: w
      type: pie
      intent: time_series
      title: W
      measures: [m]
      dimensions: [d]
      layout: { x: 0, y: 0, w: 12, h: 6 }
`
