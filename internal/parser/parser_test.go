package parser

import (
	"os"
	"path/filepath"
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
