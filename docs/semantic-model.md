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

- **entities** — join keys (`primary` or `foreign`). Required for multi-model joins later.
- **dimensions** — attributes to group or filter by. Types: `categorical`, `time`, or `number`. Time dimensions declare allowed `grains`.
- **measures** — atomic aggregations (`sum`, `count`, `count_distinct`, `avg`, `min`, `max`, `ltm`) or an `expression` over other measures (`closing - opening`). A measure may set `where` on either form. On an expression, the predicate applies to every measure the expression uses. Set `aggregation` or `expression`, not both.
- **metrics** — business metrics built from measures. Supported types in v1:
  - `simple` (default) — just the measure
  - `ratio` — numerator / denominator
  - `cumulative` — running total or grain-to-date
  - `derived` — simple expression over other metrics (restricted vocabulary; no free-form SQL)

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

Metabase uses the source table of the model that owns the card's first measure. A dimension from a second model is a left join. The join matches a shared entity name, using the foreign key on the measure model and the primary key on the other model. Both models need that entity. `-models` on a directory loads `kind: SemanticModel` files and skips a dashboard that sits in the same folder.

## Compilation targets

The SemanticModel is designed to transpile cleanly to:

- Cube YAML (`cubes:` with measures + dimensions)
- dbt Semantic Layer / MetricFlow YAML (and therefore the emerging Open Semantic Interchange / Apache Ossie format)
- Lightdash native YAML
- Metabase metrics (where the target supports them)

Dashboards continue to compile to the existing targets (Metabase, Lightdash DAC, Grafana, ECharts, Excalidraw). When a dashboard references a SemanticModel, the emitter can either inline the resolved measures or emit a reference if the target supports it.

## Design constraints (inherited from UDS)

- Minimal vocabulary — fewer concepts = fewer LLM hallucinations.
- No Jinja / templating — agents emit clean YAML.
- Honest about lossiness — document what each target cannot express.
- Version-controlled and human-reviewable.

See `schema/semantic-model.json` for the full JSON Schema and `examples/` for worked examples.
