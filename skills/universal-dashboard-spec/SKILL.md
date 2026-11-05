---
name: universal-dashboard-spec
description: "Author or edit Universal Dashboard Specification (UDS) YAML. Use when the user wants to describe a dashboard, add widgets, set intents, compile to Metabase/Excalidraw, or follow the UDS schema and taxonomy."
type: knowledge
lifecycle: active
---

# Universal Dashboard Spec — Authoring Guide

Write or edit UDS YAML that agents and the CLI can validate and compile.

## Checklist

1. Set `intent` on every widget (required for new specs).
2. Match the preferred chart type for that intent.
3. Declare measures and dimensions once; reference them by name.
4. Use the 24-column grid for layout.
5. Optionally declare `spec.source` for Metabase.

## Canonical intents

| Intent | Preferred type | Use when |
|--------|----------------|----------|
| `nominal_comparison` | `bar` | Unordered categories |
| `ranking` | `bar` | Ordered magnitude |
| `time_series` | `line`, `area` | Change over time |
| `part_to_whole` | `stacked_bar`, `pie` | Share of total |
| `deviation` | `bar`, `line` | Difference from a reference |
| `distribution` | `bar`, `heatmap` | Spread of values |
| `correlation` | `scatter`, `heatmap` | Two measures |
| `lookup` | `table`, `number` | Exact values or KPI |

Unknown intents fail validation. Non-preferred types warn (or fail with `--strict`).

## Minimal shape

```yaml
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: sales-overview
  title: Sales Overview
spec:
  source:                    # optional, used by Metabase emitter
    database: Sample Database
    schema: PUBLIC
    table: ORDERS
  measures:
    - name: revenue
      aggregation: sum
      field: amount
  dimensions:
    - name: closed_date
      type: date
      temporalGrain: month
  widgets:
    - id: revenue_over_time
      type: line
      intent: time_series
      title: Revenue over Time
      measures: [revenue]
      dimensions: [closed_date]
      layout: { x: 0, y: 0, w: 16, h: 8 }
```

## Semantic models (optional)

For reusable metrics, author a separate `kind: SemanticModel` (see the repo’s `docs/semantic-model.md` and `schema/semantic-model.json`). Dashboards can then reference measures with qualified names such as `orders.revenue`.

Inline measures on the Dashboard remain valid for simple single-table cases.

```bash
go run ./cmd/uds -in path.yaml -target excalidraw -out out.excalidraw
go run ./cmd/uds -in path.yaml -target metabase -out metabase-out
go run ./cmd/uds -in path.yaml -target metabase --strict
```

See the repo’s `docs/taxonomy.md` and `AGENTS.md` for the full rationale and CLI flags.
