package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandModelPathsSkipsDashboard(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "board.yaml"), []byte("kind: Dashboard\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "orders.yaml"), []byte("kind: SemanticModel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := expandModelPaths(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || !strings.HasSuffix(paths[0], "orders.yaml") {
		t.Fatalf("paths = %v", paths)
	}
}
