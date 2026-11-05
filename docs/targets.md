# Compilation Targets

The universal spec is intentionally smaller than any vendor format. Each transpiler maps the shared concepts into the target’s native schema.

## Metabase Representations

- Cards get stable NanoID entity IDs (from a manifest) so re-runs do not churn.
- Filters become dashboard parameters with template-tag targets.
- Goal lines map into visualization_settings.
- Lossy: Metabase measure entities have patchy serdes coverage, so measures are inlined as aggregations.

## Lightdash DAC

- Dashboard YAML + separate chart files.
- Widgets become tiles with x/y coordinates.
- metricQuery blocks for measures and dimensions.

## Grafana v2

- TabsLayout with ElementReferences.
- Raw SQL (or target-native query) compiled into each panel.
- Temporal grain is the hardest seam — Grafana’s time grouping differs from others.

## ECharts

- One TypeScript `option` object per widget.
- `markLine` for goal lines.
- Pure client-side; data is supplied separately.

## Excalidraw (visual mock)

First-class output so stakeholders can review layout, chart types, filters, and tabs without any database connection.

- Rectangles coloured by chart type.
- Tab headers.
- Filter pills with arrows showing binding.
- Grouped so visibility can be toggled per tab.

See `examples/sales-overview.excalidraw` for a worked example generated from the Sales Overview spec.
