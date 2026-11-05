# Quick start

```bash
# from the repo root
go run ./cmd/uds -in examples/sales-overview.yaml -target excalidraw -out out.excalidraw

go run ./cmd/uds -in examples/sales-overview.yaml -target metabase -out metabase-out
# SemanticModel → Cube YAML
# -models accepts files or directories (non-recursive *.yaml / *.yml).
# A directory may also hold the dashboard: files whose kind is not SemanticModel are skipped.
go run ./cmd/uds -in examples/sales-overview-semantic.yaml \
  -models examples/ \
  -target cube -out orders.cube.yaml
```

The Metabase output is a minimal Representations-shaped YAML. A full emitter would expand the MBQL queries and assign stable NanoID entity IDs.

The Cube target emits a `cubes:` document from a SemanticModel (measures, dimensions, and ratio metrics as calculated measures).

Excalidraw labels include the resolved measure title (from a SemanticModel when present, otherwise the dashboard-local title) so the mock shows the business name rather than the raw reference.
