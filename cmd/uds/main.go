package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/excalidraw"
	"github.com/cloud-shuttle/universal-dashboard-spec/internal/metabase"
	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func main() {
	var (
		input    = flag.String("in", "", "Path to UDS YAML file")
		target   = flag.String("target", "excalidraw", "Target: excalidraw | metabase")
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
		data, err := excalidraw.Emit(dash)
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
		})
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

	default:
		log.Fatalf("unknown target %q", *target)
	}
}
