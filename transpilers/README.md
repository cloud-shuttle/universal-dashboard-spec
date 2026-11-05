# Quick start

```bash
# from the repo root
go run ./cmd/uds -in examples/sales-overview.yaml -target excalidraw -out out.excalidraw

go run ./cmd/uds -in examples/sales-overview.yaml -target metabase -out metabase-out
# SemanticModel → Cube YAML
go run ./cmd/uds -in examples/sales-overview-semantic.yaml \
  -models examples/orders.semantic.yaml \
  -target cube -out orders.cube.yaml
```

The Metabase output is a minimal Representations-shaped YAML. A full emitter would expand the MBQL queries and assign stable NanoID entity IDs.

The Cube target emits a `cubes:` document from a SemanticModel (measures, dimensions, and ratio metrics as calculated measures).
