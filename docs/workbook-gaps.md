# Closing the workbook gap

UDS v1 covers most chart dashboards. These additions address the remaining third needed for workbook-style experiences.

## Parameters

User-set values that drive calculations or widget options (not just slicing).

```yaml
parameters:
  - name: period
    title: Period
    type: string
    values: [Monthly, LTM]
    affects: measure
    swaps:
      Monthly: revenue
      LTM: revenue_ltm
    default: Monthly
```

`swaps` maps each allowed value to the measure name that should be used.

## Filter controls

```yaml
filters:
  - name: recurring
    dimension: recurring_flag
    control: radio
    values: [Recurring, Non-Recurring]
```

`values` is now a defined schema property on filters.

## Series on combo charts

```yaml
series:
  - name: revenue
    type: bar
    axis: left
    color: "#1f77b4"
  - name: margin
    type: line
    axis: right
    color: "#ff7f0e"
```

## Layout units

See [layout.md](layout.md). One row unit ≈ 30px in most targets.

## LTM

`ltm` is a trailing window of the breakout grain: 12 months, 4 quarters, 52 weeks, or 365 days. A year grain is already one bucket, so it stays a single sum. Metabase emits that window with `offset`. Cube emits a 12-month rolling window, because a semantic measure has no widget grain.

## Filtered measures

`where` limits which rows an aggregation sees. It is not a dashboard filter.

```yaml
- name: lost_mrr
  aggregation: sum
  field: mrr
  where:
    - dimension: customer_status
      operator: eq
      value: Lost Customer
```

Operators: `eq`, `neq`, `in`, `not_in`, `gt`, `gte`, `lt`, `lte`. Use an array value for `in` and `not_in`.

## Expression measures

Set `expression` instead of `aggregation`. The vocabulary is measure names, numbers, `+ - * /`, parentheses, and unary minus. `where` on an expression applies to every measure it uses, so one difference can be filtered without a closing, opening, and difference helper for each movement.

```yaml
- name: lost_diff
  expression: closing - opening
  where:
    - dimension: movement
      operator: eq
      value: Lost
```

That expression field is on a measure. A SemanticModel can also name the same idea as a metric: `ratio` (numerator / denominator) or `derived` (an expression). Metabase compiles both to MBQL. Cube compiles `ratio` only. `cumulative` stays in the schema and is not compiled.

## Numeric dimensions

Semantic dimensions accept `type: number` alongside `categorical` and `time`.

## Filter reuse

A filter sets `dimension` or `bindings`. Bindings are dimension names or `model.dimension` names, so one control can slice the same idea on more than one model.

```yaml
- name: segment
  control: multi_select
  searchable: true
  bindings: [orders.segment, customers.segment]
```

`searchable: true` asks the target for a search box on that control. It does not change `control`.

## Time grain

`timeGrain` on a widget overrides that widget's time dimension grain (`day`, `week`, `month`, `quarter`, `year`).

## Tab title templates

Tabs and widgets accept `titleTemplate`, same `{{parameter}}` form. Metabase substitutes the parameter default into the tab name and the card name.

## Static widgets

`text` requires `content`. `image` requires `src`. Omit `intent` on both.

## Fixed filters

`fixedFilters` on a widget are constant predicates, using the same shape as measure `where`. A per-year KPI tile sets `operator: eq` on the year dimension.

## Pivot order

`rows` and `columns` are ordered lists. Each entry is `{ dimension: name }` or `{ measure: name }`. A measure stays in the bucket where it is listed.

Pivot formatting lives on `options`:

```yaml
options:
  boldTotals: true
  negatives: brackets
  conditionalColors:
    - measure: lost_diff
      when: negative
      color: "#e8590c"
```

`when` is `negative` or `positive`. Each dimension in `columns` is one grouped-header level. Metabase records `negative_in_parentheses` on each measure. Its number formatter still prints a leading minus.

## Still under x- keys

Query structure is in the schema. These presentation and interaction details are not, so they stay on `x-` keys: tooltip fields beyond `series.tooltipFields`, selected-cell highlighting, charts that stay empty until a click, footer and extra rows, the percent-of-total line, and project notes. Count colours that are a positive or negative rule can use `options.conditionalColors`.

## Status

| Feature                       | Schema | Parser | Metabase emitter |
|-------------------------------|--------|--------|------------------|
| Parameters + values + swaps   | yes    | yes    | partial          |
| Radio + filter values         | yes    | yes    | yes              |
| Dynamic titles                | yes    | yes    | yes              |
| Combo series type/axis        | yes    | yes    | `series_settings` display, axis, and colour |
| Series colours                | yes    | yes    | yes              |
| Pivot formatting              | yes    | yes    | column split, totals, colours; brackets recorded |
| LTM (dashboard + semantic)    | yes    | yes    | trailing window via `offset` |
| Layout units documented       | yes    | —      | —                |
| Measure `where`               | yes    | yes    | `sum-where` / `count-where` / `distinct-where` |
| Measure `expression`          | yes    | yes    | MBQL `+ - * /`   |
| Semantic `number` dimension   | yes    | yes    | —                |
| Filter `bindings`             | yes    | yes    | one parameter, per-card mapping; skipped bindings warn |
| Filter `searchable`           | yes    | yes    | passed on the parameter |
| Widget `timeGrain`            | yes    | yes    | `temporal-unit`  |
| Tab `titleTemplate`           | yes    | yes    | tab and card name, `{{param}}` from the parameter default |
| `text` / `image` widgets      | yes    | yes    | virtual text card |
| Widget `fixedFilters`         | yes    | yes    | query `filter`   |
| Pivot `rows` / `columns`      | yes    | yes    | measures stay in the listed bucket |
| Semantic metric `simple` / `ltm` | yes | yes  | the measure, or an `offset` window |
| Semantic metric `ratio` / `derived` | yes | yes | MBQL expression |
| Semantic metric `cumulative`  | yes    | yes    | skipped, with a warning |
| Model source                  | yes    | yes    | model table, else dashboard source, else Sample Database / PUBLIC / ORDERS, with a warning |
| Model join                    | yes    | yes    | left join only for a query field; one side must be `primary`; a filter does not join |
| `-database` / `-schema` / `-table` | — | — | `-database` on every table; `-schema` and `-table` on the base table |
| Widget `actions`              | yes    | yes    | not emitted, with a warning |
| Parameter `swaps`             | yes    | yes    | stored on the parameter; the card measure is not swapped, with a warning |

These are additive. Existing specs remain valid. Lightdash and Grafana have no emitter. ECharts, Excalidraw, and Cube coverage is in [targets.md](targets.md) and [cross-target.md](cross-target.md).
