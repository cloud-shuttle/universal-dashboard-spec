package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/cube"
	"github.com/cloud-shuttle/universal-dashboard-spec/internal/echarts"
	"github.com/cloud-shuttle/universal-dashboard-spec/internal/excalidraw"
	"github.com/cloud-shuttle/universal-dashboard-spec/internal/metabase"
	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func main() {
	var (
		input    = flag.String("in", "", "Path to UDS Dashboard YAML file")
		models   = flag.String("models", "", "Comma-separated paths to SemanticModel YAML files or directories containing them")
		target   = flag.String("target", "excalidraw", "Target: excalidraw | metabase | echarts | cube")
		out      = flag.String("out", "", "Output path (directory or file)")
		strict   = flag.Bool("strict", false, "Treat intent/type mismatches as errors")
		database = flag.String("database", "", "Override source database (Metabase)")
		schema   = flag.String("schema", "", "Override source schema (Metabase)")
		table    = flag.String("table", "", "Override source table (Metabase)")
	)
	flag.Parse()

	if *input == "" {
		log.Fatal("missing -in")
	}

	dash, err := parser.Load(*input)
	if err != nil {
		log.Fatal(err)
	}

	reg := parser.NewRegistry()
	if *models != "" {
		paths, err := expandModelPaths(*models)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range paths {
			m, err := parser.LoadSemanticModel(p)
			if err != nil {
				log.Fatalf("load model %s: %v", p, err)
			}
			if err := m.Validate(); err != nil {
				log.Fatalf("model %s: %v", p, err)
			}
			reg.Add(m)
			fmt.Fprintf(os.Stderr, "loaded model %s from %s\n", m.Metadata.Name, p)
		}
	}

	warnings, err := dash.Validate()
	if err != nil {
		log.Fatal(err)
	}
	if *strict && len(warnings) > 0 {
		for _, w := range warnings {
			fmt.Fprintln(os.Stderr, "error:", w)
		}
		log.Fatal("strict mode: intent/type mismatches")
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}

	switch *target {
	case "excalidraw":
		data, err := excalidraw.Emit(dash, reg)
		if err != nil {
			log.Fatal(err)
		}
		path := *out
		if path == "" {
			path = "out.excalidraw"
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", path)

	case "metabase":
		dir := *out
		if dir == "" {
			dir = "metabase-out"
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
		files, err := metabase.Emit(dash, metabase.EmitOptions{
			Database: *database,
			Schema:   *schema,
			Table:    *table,
		}, reg)
		if err != nil {
			log.Fatal(err)
		}
		for name, content := range files {
			p := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				log.Fatal(err)
			}
			if err := os.WriteFile(p, content, 0644); err != nil {
				log.Fatal(err)
			}
			fmt.Println("wrote", p)
		}

	case "echarts":
		dir := *out
		if dir == "" {
			dir = "echarts-out"
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
		files, err := echarts.Emit(dash)
		if err != nil {
			log.Fatal(err)
		}
		for name, content := range files {
			p := filepath.Join(dir, name)
			if err := os.WriteFile(p, content, 0644); err != nil {
				log.Fatal(err)
			}
			fmt.Println("wrote", p)
		}

	case "cube":
		if len(reg.Models()) == 0 {
			log.Fatal("cube target requires -models pointing at a SemanticModel or directory")
		}
		// Emit the first loaded model
		var first *parser.SemanticModel
		for _, m := range reg.Models() {
			first = m
			break
		}
		data, err := cube.Emit(first)
		if err != nil {
			log.Fatal(err)
		}
		path := *out
		if path == "" {
			path = first.Metadata.Name + ".cube.yaml"
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", path)

	default:
		log.Fatalf("unknown target %q", *target)
	}
}

// expandModelPaths turns a comma-separated list of files and/or directories
// into a flat list of SemanticModel YAML paths. Directories are walked
// non-recursively for *.yaml / *.yml files.
func expandModelPaths(raw string) ([]string, error) {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", p, err)
		}
		if !info.IsDir() {
			out = append(out, p)
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil, fmt.Errorf("read dir %s: %w", p, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
				out = append(out, filepath.Join(p, name))
			}
		}
	}
	return out, nil
}
