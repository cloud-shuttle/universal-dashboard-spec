package metabase

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func TestStableIDsRoundTrip(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "sales-overview.yaml")
	dash, err := parser.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	reg := parser.NewRegistry()

	first, err := Emit(dash, EmitOptions{}, reg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Emit(dash, EmitOptions{}, reg)
	if err != nil {
		t.Fatal(err)
	}

	if len(first) != len(second) {
		t.Fatalf("file count differs: %d vs %d", len(first), len(second))
	}

	for name, content := range first {
		other, ok := second[name]
		if !ok {
			t.Fatalf("second emit missing file %s", name)
		}
		if string(content) != string(other) {
			t.Fatalf("content differs for %s on re-emit", name)
		}
	}
}

func TestStableIDDeterministic(t *testing.T) {
	a := stableID("card-revenue_over_time")
	b := stableID("card-revenue_over_time")
	if a != b {
		t.Fatalf("stableID not deterministic: %s vs %s", a, b)
	}
	if len(a) != 21 {
		t.Fatalf("stableID length = %d, want 21", len(a))
	}

	// Different inputs produce different IDs
	c := stableID("card-deals_by_region")
	if a == c {
		t.Fatal("different inputs produced the same ID")
	}
}

func TestStableIDsSurviveReload(t *testing.T) {
	// Write the example to a temp file, load twice, emit, compare entity IDs
	src, err := os.ReadFile(filepath.Join("..", "..", "examples", "sales-overview.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "dash.yaml")
	if err := os.WriteFile(p, src, 0644); err != nil {
		t.Fatal(err)
	}

	d1, err := parser.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := parser.Load(p)
	if err != nil {
		t.Fatal(err)
	}

	out1, err := Emit(d1, EmitOptions{}, parser.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	out2, err := Emit(d2, EmitOptions{}, parser.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}

	for name, content := range out1 {
		if string(content) != string(out2[name]) {
			t.Fatalf("reload produced different output for %s", name)
		}
	}
}
