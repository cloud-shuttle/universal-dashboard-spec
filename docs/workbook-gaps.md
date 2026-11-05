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

These are additive. Existing specs remain valid.
