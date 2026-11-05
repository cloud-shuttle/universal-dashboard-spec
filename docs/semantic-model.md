# Semantic Model (Metrics Layer)

UDS supports a first-class **SemanticModel** resource that acts as a reusable metrics / semantic layer. Dashboards can reference it instead of (or in addition to) defining measures and dimensions inline.

This keeps the dashboard YAML thin and presentation-focused while giving agents and humans a single source of truth for business metrics.

## Why a separate kind?

| Concern | Dashboard (current) | SemanticModel (new) |
|---------|---------------------|---------------------|
| Scope | One dashboard’s layout and intent | Reusable metrics and dimensions across many consumers |
| Measures | Simple aggregations over a field | Atomic measures + derived metrics (ratio, cumulative, …) |
| Dimensions | Local to the dashboard | Shared, with time grains and entity relationships |
| Reuse | Copy-paste | Reference by name |
| Targets | Metabase, Grafana, Excalidraw, … | Cube, dbt Semantic Layer / MetricFlow, Lightdash, OSI, … |

A SemanticModel is optional. Existing dashboards with inline measures continue to work.

## Minimal example

```yaml
apiVersion: uds/v1
kind: SemanticModel
metadata:
  name: orders
  title: Orders
  description: Core order facts

spec:
  source:
    database: analytics
    schema: marts
    table: fct_orders

  entities:
    - name: order
      type: primary
      field: order_id
    - name: customer
      type: foreign
      field: customer_id

  dimensions:
    - name: ordered_at
      type: time
      field: ordered_at
      grains: [day, week, month, quarter, year]
    - name: region
      type: categorical
      field: region

  measures:
    - name: revenue
      aggregation: sum
      field: amount
    - name: order_count
      aggregation: count
      field: order_id

  metrics:
    - name: avg_order_value
      type: ratio
      numerator: revenue
      denominator: order_count
```

## Core concepts (kept deliberately small)

- **entities** — join keys (`primary` or `foreign`). Metabase joins two models on a shared entity name only when one side is `primary`.
- **dimensions** — attributes to group or filter by. Types: `categorical`, `time`, or `number`. Time dimensions declare allowed `grains`.
- **measures** — atomic aggregations (`sum`, `count`, `count_distinct`, `avg`, `min`, `max`, `ltm`) or an `expression` over other measures (`closing - opening`). A measure may set `where` on either form. On an expression, the predicate applies to every measure the expression uses. Set `aggregation` or `expression`, not both.
- **metrics** — business metrics built from measures. Supported types in v1:
  - `simple` — the named measure. Metabase emits that measure.
  - `ltm` — a trailing window of the named measure. Metabase emits the same `offset` window as a measure `ltm`. Cube does not emit metric `ltm`. A measure with `aggregation: ltm` is what Cube turns into a 12-month `rolling_window`.
  - `ratio` — numerator / denominator. Metabase emits MBQL `/`. Cube emits `{numerator} / {denominator}`.
  - `derived` — an expression over measures or other metrics, same vocabulary as a measure expression. Metabase emits MBQL `+ - * /`. Cube does not emit it.
  - `cumulative` — running total or grain-to-date (`input_metric`, `grain_to_date`). The schema accepts it. Metabase skips it and prints a warning. Cube does not emit it.

## Referencing from a Dashboard

```yaml
apiVersion: uds/v1
kind: Dashboard
metadata:
  name: sales-overview
  title: Sales Overview
spec:
  models: [orders]          # optional
  widgets:
    - id: revenue_over_time
      type: line
      intent: time_series
      title: Revenue over Time
      measures: [orders.revenue]
      dimensions: [orders.ordered_at]
      layout: { x: 0, y: 0, w: 16, h: 8 }
```

Qualified names (`model.measure`) are resolved against loaded SemanticModels. Unqualified names still work for inline definitions.

Metabase uses the source table of the model that owns the card's first measure. A second model is joined only when a query field needs it (a breakout, pivot field, fixed filter, or measure `where`). One side of the shared entity must be a primary key. A dashboard filter does not add a join, and a filter whose model is not in the query is left unmapped. A model with no source, database, or table is filled from the dashboard source, then `Sample Database` / `PUBLIC` / `ORDERS`, and the emitter warns. `-database` sets the database on the base table and on every join. `-models` on a directory loads `kind: SemanticModel` files and skips a dashboard that sits in the same folder.

## Compilation targets

Implemented in this repo:

- **Cube** (`-target cube`) writes `cubes:` with dimensions, measures, measure expressions, measure `where`, measure `ltm`, and `ratio` metrics. The CLI emits the first model given to `-models`.
- **Metabase** inlines the resolved measure or metric into the card's MBQL aggregation. It does not emit a separate Metabase metric entity. Joins and source rules are in the section above.
- **Excalidraw** shows the SemanticModel measure or metric title on the widget label when `-models` loaded that model.
- **ECharts** does not read the registry. Qualified names stay as the widget wrote them, and the series data is a placeholder.

Designed so a later emitter can target them, and not implemented here: dbt Semantic Layer / MetricFlow, Lightdash native YAML, and Grafana.

## Design constraints (inherited from UDS)

- Minimal vocabulary — fewer concepts = fewer LLM hallucinations.
- No Jinja / templating — agents emit clean YAML.
- Honest about lossiness — document what each target cannot express.
- Version-controlled and human-reviewable.

See `schema/semantic-model.json` for the full JSON Schema and `examples/` for worked examples.
