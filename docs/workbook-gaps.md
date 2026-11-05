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

`ltm` is a valid measure aggregation and a valid metric type in both dashboards and semantic models.

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

Set `expression` instead of `aggregation`. The vocabulary is measure names, numbers, `+ - * /`, parentheses, and unary minus.

```yaml
- name: mrr_change
  expression: mrr - prev_mrr
```

Semantic metrics of `type: derived` are unchanged. This field is for measures.

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

Tabs accept `titleTemplate`, same `{{parameter}}` form as widget titles.

## Static widgets

`text` requires `content`. `image` requires `src`. Omit `intent` on both.

## Fixed filters

`fixedFilters` on a widget are constant predicates, using the same shape as measure `where`. A per-year KPI tile sets `operator: eq` on the year dimension.

## Pivot order

`rows` and `columns` are ordered lists. Each entry is `{ dimension: name }` or `{ measure: name }`.

## Status

| Feature                       | Schema | Parser | Metabase emitter |
|-------------------------------|--------|--------|------------------|
| Parameters + values + swaps   | yes    | yes    | partial          |
| Radio + filter values         | yes    | yes    | yes              |
| Dynamic titles                | yes    | yes    | yes              |
| Combo series type/axis        | yes    | yes    | partial          |
| Series colours                | yes    | yes    | yes              |
| Pivot formatting              | yes    | yes    | rows/columns only|
| LTM (dashboard + semantic)    | yes    | yes    | emitted as sum   |
| Layout units documented       | yes    | —      | —                |
| Measure `where`               | yes    | yes    | `sum-where` / `count-where` / `distinct-where` |
| Measure `expression`          | yes    | yes    | MBQL `+ - * /`   |
| Semantic `number` dimension   | yes    | yes    | —                |
| Filter `bindings`             | yes    | yes    | one parameter, per-card mapping |
| Filter `searchable`           | yes    | yes    | passed on the parameter |
| Widget `timeGrain`            | yes    | yes    | `temporal-unit`  |
| Tab `titleTemplate`           | yes    | yes    | not emitted (no tabs yet) |
| `text` / `image` widgets      | yes    | yes    | virtual text card |
| Widget `fixedFilters`         | yes    | yes    | query `filter`   |
| Pivot `rows` / `columns`      | yes    | yes    | row order stored; measures go to values |

These are additive. Existing specs remain valid.
