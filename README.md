# Universal Dashboard Specification (UDS)

**Status: experimental.** The schema and the `uds` CLI compile a dashboard to Metabase, ECharts, and Excalidraw, and a SemanticModel to Cube. The format can still change. Pin to a commit if you depend on it.

Lightdash and Grafana are described as targets. This repo has no emitter for them. Drill actions (`actions` on a widget) are in the schema and are not emitted. A `cumulative` metric is accepted by the schema and is not compiled.

A thin, LLM-friendly YAML language for describing dashboards once, then compiling that description to multiple targets.

## Goals

- **Single source of intent** — measures, dimensions, filters, layout, chart types, and analytical messages live in one place.
- **LLM-authorable** — deliberately smaller and more regular than any vendor format so agents get it right.
- **Human-reviewable** — readable by people and renderable as an Excalidraw mock before any data is connected.
- **Multi-target** — compile to Metabase, ECharts, Excalidraw, and Cube today. Lightdash and Grafana are specified as later targets.
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

**LLMs:** read [AGENTS.md](AGENTS.md) for the short authoring checklist, or install the skill from [skills/universal-dashboard-spec](skills/universal-dashboard-spec).

## High-level structure

Dashboards:

```yaml
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: sales-overview
  title: Sales Overview
  description: Revenue and pipeline by region and product
spec:
  source: { database, schema, table }   # optional; Metabase
  models: [orders]                      # optional SemanticModel names
  parameters: [...]
  measures: [...]
  dimensions: [...]
  filters: [...]
  tabs: [...]
  widgets: [...]
```

Semantic models (metrics layer):

```yaml
apiVersion: uds/v1
kind: SemanticModel
metadata:
  name: orders
  title: Orders
spec:
  source: { database, schema, table }
  entities: [...]
  dimensions: [...]
  measures: [...]
  metrics: [...]
```

See `schema/dashboard.json` and `schema/semantic-model.json` for the full JSON Schemas.

## Docs

| Doc | What it covers |
|-----|----------------|
| [docs/taxonomy.md](docs/taxonomy.md) | Intents and the preferred chart type for each |
| [docs/layout.md](docs/layout.md) | The 24-column grid |
| [docs/semantic-model.md](docs/semantic-model.md) | Measures, metrics, entities, and joins |
| [docs/workbook-gaps.md](docs/workbook-gaps.md) | Parameters, filters, pivots, LTM, expressions, and the emitter status table |
| [docs/targets.md](docs/targets.md) | What each compiler writes, and what it drops |
| [docs/cross-target.md](docs/cross-target.md) | The same concept across targets |
| [transpilers/README.md](transpilers/README.md) | CLI flags and commands |
| [AGENTS.md](AGENTS.md) | Short checklist for authors |

## What a file can say

A dashboard may set `source`, `models`, `parameters`, `measures`, `dimensions`, `filters`, `tabs`, and `widgets`. Measures and dimensions can live on the dashboard or on a SemanticModel referenced as `model.name`.

Shipped vocabulary, with the YAML in [docs/workbook-gaps.md](docs/workbook-gaps.md) and [docs/semantic-model.md](docs/semantic-model.md):

- Measure `where` and measure `expression` (`+ - * /`, parentheses, unary minus).
- Semantic metrics: `simple`, `ltm`, `ratio`, `derived`, and `cumulative` (schema only).
- Entities (`primary` or `foreign`) so Metabase can join a second model.
- Filter `bindings`, `values`, `control` (including `radio`), and `searchable`.
- Parameter `values`, `default`, and `swaps`.
- Widget `intent`, `timeGrain`, `titleTemplate`, `fixedFilters`, `parameters`, `series` (`type`, `axis`, `color`, `tooltipFields`), and pivot `rows` / `columns`.
- Widget types `text` (`content`) and `image` (`src`). Omit `intent` on those two.
- Options used by Metabase: `goalLine`, `boldTotals`, `negatives: brackets`, `conditionalColors`.
- Tab `titleTemplate`.

`examples/sales-overview.yaml` is the small chart dashboard. `examples/workbook.yaml` uses the workbook fields. `examples/orders.semantic.yaml` is a SemanticModel. `examples/sales-overview-semantic.yaml` is a dashboard that references one.

## Try the CLI

```bash
go run ./cmd/uds -in examples/sales-overview.yaml -target excalidraw -out out.excalidraw
go run ./cmd/uds -in examples/sales-overview.yaml -target metabase -out metabase-out
go run ./cmd/uds -in examples/sales-overview.yaml -target echarts -out echarts-out
go run ./cmd/uds -in examples/sales-overview-semantic.yaml \
  -models examples/ -target cube -out orders.cube.yaml
```

| Flag | Meaning |
|------|---------|
| `-in` | Dashboard YAML. Required. |
| `-target` | `excalidraw` (default), `metabase`, `echarts`, or `cube`. |
| `-out` | File for Excalidraw and Cube. Directory for Metabase and ECharts. |
| `-models` | Comma-separated SemanticModel files or directories. A directory is scanned one level for `*.yaml` / `*.yml`. Files whose `kind` is not `SemanticModel` are skipped. |
| `-strict` | Intent/type mismatches fail the run. Otherwise they are warnings. |
| `-database`, `-schema`, `-table` | Metabase source override. `-database` sets the database on the base table and on every join. `-schema` and `-table` set the base table only. |

`go test ./...` runs the parser and emitter tests. Flag detail and per-target behaviour are in [transpilers/README.md](transpilers/README.md) and [docs/targets.md](docs/targets.md).
