# Universal Dashboard Specification (UDS)

A thin, LLM-friendly YAML language for describing dashboards once, then compiling that description to multiple targets.

## Goals

- **Single source of intent** — measures, dimensions, filters, layout, and chart types live in one place.
- **LLM-authorable** — deliberately smaller and more regular than any vendor format so agents get it right.
- **Human-reviewable** — readable by people and renderable as an Excalidraw mock before any data is connected.
- **Multi-target** — compile to Metabase, Lightdash, Grafana, ECharts, and more.
- **No dbt / Jinja** — in the age of LLMs we do not need a templating language. Agents emit clean SQL or the target’s native query shape when required.

## Design Principles

1. Semantic over syntactic — declare what you want measured and sliced, not the SQL.
2. Minimal vocabulary — fewer concepts = fewer ways for an LLM to hallucinate.
3. Explicit layout — 24-column grid so position is deterministic.
4. Honest about lossiness — document what does not survive compilation to each target.
5. Visual review gate — Excalidraw output is a first-class deliverable.

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
