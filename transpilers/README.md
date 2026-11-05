# Quick start

```bash
# from the repo root
go run ./cmd/uds -in examples/sales-overview.yaml -target excalidraw -out out.excalidraw

go run ./cmd/uds -in examples/sales-overview.yaml -target metabase -out metabase-out
```

The Metabase output is a minimal Representations-shaped YAML. A full emitter would expand the MBQL queries and assign stable NanoID entity IDs.
