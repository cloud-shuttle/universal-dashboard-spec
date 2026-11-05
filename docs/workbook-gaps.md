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
    default: Monthly
  - name: currency
    title: Currency View
    type: string
    values: [Reported, Constant]
    affects: measure
```

`affects` says what the parameter changes: `measure`, `dimension`, `title`, `filter`, or `option`.

## Filter controls

```yaml
filters:
  - name: recurring
    dimension: recurring_flag
    control: radio
    values: [Recurring, Non-Recurring]
  - name: region_filter
    dimension: region
    control: multi_select
    multi: true
```

`control` now includes `radio`.

## Dynamic titles

```yaml
- id: arr_trend
  title: Monthly ARR
  titleTemplate: "{{period}} ARR"
  parameters: [period]
```

## Combo and diverging charts

New widget types: `combo` and `diverging`.

## Series styling

```yaml
series:
  - name: revenue
    color: "#1f77b4"
    tooltipFields: [deal_count, avg_deal_size]
```

## Pivot formatting

Use `options` on a `pivot` widget:

```yaml
options:
  rows: [region]
  columns: [product]
  groupedHeaders: true
  boldTotals: true
  bracketNegatives: true
  conditionalColors:
    - field: revenue
      when: "< 0"
      color: "#d62728"
```

## LTM

`ltm` is a valid measure aggregation (last-twelve-months). Semantic models can also declare an LTM metric type.

## Status

| Feature                       | Schema | Parser | Metabase emitter |
|-------------------------------|--------|--------|------------------|
| Parameters + values + affects | yes    | yes    | partial          |
| Radio control                 | yes    | yes    | yes (as string/=)|
| Dynamic titles                | yes    | yes    | yes (titleTemplate)|
| Combo / diverging             | yes    | yes    | partial          |
| Series colours + tooltips     | yes    | yes    | colours only     |
| Pivot formatting              | yes    | yes    | rows/columns only|
| LTM aggregation               | yes    | yes    | emitted as sum   |

These are additive. Existing specs remain valid. The Metabase emitter now consumes the new fields where the target can express them; other emitters still need the same treatment.
