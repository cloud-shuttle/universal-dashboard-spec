package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_MetabaseHasParameterMappings(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "mb")

	cmd := exec.Command("go", "run", "./cmd/uds",
		"-in", "examples/sales-overview.yaml",
		"-target", "metabase",
		"-out", out,
	)
	cmd.Dir = repoRoot(t)
	outb, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli failed: %v\n%s", err, outb)
	}

	// Find the dashboard YAML
	var dash []byte
	err = filepath.Walk(out, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(p, "dashboards") && strings.HasSuffix(p, ".yaml") {
			dash, err = os.ReadFile(p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if dash == nil {
		t.Fatal("no dashboard YAML produced")
	}
	s := string(dash)
	if !strings.Contains(s, "parameter_mappings") {
		t.Fatal("dashboard missing parameter_mappings")
	}
	if !strings.Contains(s, "date_range") && !strings.Contains(s, "template-tag") {
		t.Fatal("parameter mappings do not reference filters")
	}
}

func TestCLI_EChartsHasSeries(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "ech")

	cmd := exec.Command("go", "run", "./cmd/uds",
		"-in", "examples/sales-overview.yaml",
		"-target", "echarts",
		"-out", out,
	)
	cmd.Dir = repoRoot(t)
	outb, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli failed: %v\n%s", err, outb)
	}

	// revenue_over_time.ts should exist and contain a series
	p := filepath.Join(out, "revenue_over_time.ts")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "series") {
		t.Fatal("echarts output missing series")
	}
	if !strings.Contains(s, ""type": "line"") && !strings.Contains(s, "\"type\": \"line\"") {
		// JSON marshal uses no spaces around colon in some cases; accept either
		if !strings.Contains(s, "line") {
			t.Fatal("expected line series in revenue_over_time.ts")
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// Tests run from the package dir; walk up to the module root
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}
