# Universal Dashboard Specification (UDS)

A thin, LLM-friendly YAML language for describing dashboards once, then compiling that description to multiple targets.

## Goals

- **Single source of intent** — measures, dimensions, filters, layout, chart types, and analytical messages live in one place.
- **LLM-authorable** — deliberately smaller and more regular than any vendor format so agents get it right.
- **Human-reviewable** — readable by people and renderable as an Excalidraw mock before any data is connected.
- **Multi-target** — compile to Metabase, Lightdash, Grafana, ECharts, and more.
- **No dbt / Jinja** — in the age of LLMs we do not need a templating language. Agents emit clean SQL or the target’s native query shape when required.

## Design Principles

1. Semantic over syntactic — declare what you want measured and sliced, and the message you want to convey, not the SQL.
2. Minimal vocabulary — fewer concepts = fewer ways for an LLM to hallucinate.
3. Explicit layout — 24-column grid so position is deterministic.
4. Honest about lossiness — document what does not survive compilation to each target.
5. Visual review gate — Excalidraw output is a first-class deliverable.

## Message taxonomy

Widgets can declare an optional `intent` (analytical message) drawn from a small canonical set. This makes the spec more robust for LLMs, validation, and multi-target compilation.

See [docs/taxonomy.md](docs/taxonomy.md) for the full list, preferred encodings, and rationale (based on Stephen Few’s quantitative messages and common BI practice).

## High-level structure

```yaml
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: sales-overview
  title: Sales Overview
  description: Revenue and pipeline by region and product
spec:
  measures: [...]
  dimensions: [...]
  filters: [...]
  tabs: [...]
  widgets: [...]
```

See `schema/dashboard.json` for the full JSON Schema and `examples/` for worked examples.

## Try the CLI

```bash
go run ./cmd/uds -in examples/sales-overview.yaml -target excalidraw -out out.excalidraw
go run ./cmd/uds -in examples/sales-overview.yaml -target metabase -out metabase-out
```

See `transpilers/README.md` for details. The Metabase emitter is minimal (layout + parameters); a full version would emit MBQL and stable NanoIDs.
