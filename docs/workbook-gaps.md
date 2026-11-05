# Closing the workbook gap

UDS v1 covers most chart dashboards. These additions address the remaining third needed for workbook-style experiences.

## Parameters

User-set values that drive calculations or widget options (not just slicing).

```yaml
parameters:
  - name: target_margin
    title: Target Margin
    type: number
    default: 0.3
  - name: as_of_date
    title: As Of
    type: date
```

Widgets declare which parameters they consume:

```yaml
widgets:
  - id: margin_vs_target
    parameters: [target_margin]
```

## Filter controls

Existing filters gain UI behaviour.

```yaml
filters:
  - name: region_filter
    dimension: region
    control: multi_select
    multi: true
  - name: date_range
    dimension: closed_date
    control: date_range
    required: true
  - name: product_filter
    dimension: product
    control: dropdown
    cascadeFrom: region_filter
```

`control` values: `dropdown`, `multi_select`, `date_picker`, `date_range`, `slider`, `search`.

## Pivot tables

New widget type.

```yaml
- id: sales_pivot
  type: pivot
  title: Sales by Region and Product
  measures: [revenue]
  dimensions: [region, product]
  options:
    rows: [region]
    columns: [product]
```

## Drill actions

```yaml
- id: deals_by_region
  actions:
    - type: drill_to_dashboard
      target: region-detail
      parameterMappings:
        region: region
    - type: url
      target: "https://example.com/deals?region={{region}}"
```

## Status

| Feature            | Schema | Parser | Emitters      |
|--------------------|--------|--------|---------------|
| Parameters         | yes    | pending| pending       |
| Filter controls    | yes    | pending| pending       |
| Pivot tables       | yes    | pending| pending       |
| Drill actions      | yes    | pending| pending       |

These are additive. Existing specs remain valid.
