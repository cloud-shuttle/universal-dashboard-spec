# Compilation Targets

The universal spec is intentionally smaller than any vendor format. Each transpiler maps the shared concepts into the target’s native schema.

## Metabase Representations (implemented)

Output is one card YAML per analytical widget and one dashboard YAML, under `collections/main/`. Entity ids are a 21-character hash of the name (NanoID alphabet).

- The query is MBQL. Measures are inlined as aggregations. `where` on `sum`, `count`, and `count_distinct` becomes `sum-where`, `count-where`, or `distinct-where`. `avg`, `min`, and `max` have no `*-where` form, so the predicate is a query filter only when the card has one measure.
- An expression is MBQL `+ - * /`. `where` on an expression applies to every measure it uses.
- `graph.dimensions` uses each dimension's `field`. `graph.metrics` uses the short measure name. `ranking` adds `order-by` descending on the first aggregation. `part_to_whole` and `stacked_bar` set `graph.stacking`. `options.goalLine` sets `graph.goal_value`.
- Series `color` becomes `graph.colors`. Series `type` and `axis` are not written on the Metabase card.
- A pivot writes `pivot_table.column_split`. A measure stays in the `rows` or `columns` bucket where it is listed. `boldTotals` sets `pivot.show_row_totals` and `pivot.show_column_totals`. `conditionalColors` becomes `table.column_formatting`. `negatives: brackets` records `negative_in_parentheses` per measure. Metabase's formatter still prints a leading minus.
- `ltm` is an `offset` window of the breakout grain: 12 months, 4 quarters, 52 weeks, 365 days, or one sum for a year.
- Semantic metrics: `simple` is the named measure, `ltm` is that window, `ratio` is numerator `/` denominator, `derived` is its expression. `cumulative` is skipped and a warning is printed.
- The card table is the measure model's `source`, then the dashboard `source`, then `Sample Database` / `PUBLIC` / `ORDERS`. A model that needed a default prints a warning. `-database` sets the database on the base table and every join. `-schema` and `-table` set the base table only.
- A second model is a left join when a query field needs it (measure, breakout, pivot field, fixed filter, or measure `where`). One side of the shared entity must be `primary`. Two foreign keys fail the card. A filter binding does not add a join. A filter whose model is not on the card is left unmapped. The mapping target is a field ref, including `join-alias` when the field's model was joined.
- Filters and UDS parameters become dashboard parameters. `searchable`, static `values`, and `bindings` are passed through. Radio and multi-select use `string/=`. Date controls use `date/range` or `date/single`.
- `titleTemplate` substitutes `{{parameter}}` from the parameter `default` into the tab name and the card name. The tab also keeps the raw string as `title_template`. Widget `parameters` do not become template tags.
- `fixedFilters` are the query `filter`. `timeGrain` sets `temporal-unit` on the time breakout.
- `text` and `image` are virtual text cards on the dashboard. An image card's body is `src`. Layout is `col` / `row` / `size_x` / `size_y`.
- Unsafe YAML scalars (`=`, `-`, `/`, and the rest of that set) are double-quoted.
- Not emitted: widget `actions`, parameter `swaps` as a measure change, series `type` and `axis`, `cumulative` metrics.

## Cube (implemented)

`uds -target cube` writes one `cubes:` document from the first SemanticModel passed with `-models`.

- Dimensions, including `number`. Time stays `time`. Categorical becomes `string`.
- Measures. An expression is `type: number` with `{measure}` refs. `where` becomes Cube `filters`. A measure with `aggregation: ltm` is `type: sum` plus `rolling_window: {trailing: "12 month", offset: end}`.
- A `ratio` metric is a calculated measure, `{numerator} / {denominator}`.
- Not emitted: `derived`, `cumulative`, and metric `type: ltm`. Cube has no widget grain, so the rolling window is fixed at 12 months. Dashboard layout, filters, and widgets are ignored.

## Lightdash DAC (not implemented)

Described here so a later emitter has a target shape. Nothing in `cmd/uds` writes Lightdash.

- Dashboard YAML + separate chart files.
- Widgets become tiles with x/y coordinates.
- metricQuery blocks for measures and dimensions.

## Grafana v2 (not implemented)

Described here so a later emitter has a target shape. Nothing in `cmd/uds` writes Grafana.

- TabsLayout with ElementReferences.
- Raw SQL (or target-native query) compiled into each panel.
- Temporal grain is the hardest seam. Grafana's time grouping differs from the other targets.

## ECharts (implemented)

- One TypeScript `option` per widget (`internal/echarts`). Data is a placeholder.
- Series `type`, `axis` (`yAxisIndex` for `right`), `color`, and `tooltipFields`. A combo gets a left and a right value axis.
- `titleTemplate` is copied raw. It is not resolved from the parameter default. Widget `parameters` are attached as `udsParameters`, including `swaps`.
- `timeGrain` is the category axis name. `options.goalLine` is a `markLine`.
- `text` and `image` export `udsType` plus `content` or `src`, and no series.
- Supported chart types: line, area, bar, stacked_bar, combo, pie, scatter, number (gauge). Other types fall through to a generic series.

```bash
go run ./cmd/uds -in examples/sales-overview.yaml -target echarts -out echarts-out
```

## Excalidraw (implemented)

A mock so layout, chart types, filters, and tabs can be reviewed with no database.

- Rectangles coloured by chart type. Position comes from the 24-column grid (32px per column, 36px per row).
- Tab headers. A tab uses `titleTemplate` when set, otherwise `title`. The template is not resolved.
- Filter pills. The label includes the control, and `+search` when `searchable` is set.
- Parameter pills. The label is the title, plus `affects` in parentheses when that field is set. `swaps` is not drawn.
- Widget label: type, intent, title or raw `titleTemplate`, `timeGrain`, fixed filters, series type and axis, and measure titles. With `-models`, a qualified measure or metric uses the SemanticModel title.
- `text` and `image` get their own colours.

`examples/sales-overview.excalidraw` is a mock generated from the Sales Overview spec. The current emitter does not draw filter-binding arrows or per-tab groups.
