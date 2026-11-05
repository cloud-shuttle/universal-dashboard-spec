# CLI

```bash
go run ./cmd/uds -in examples/sales-overview.yaml -target excalidraw -out out.excalidraw
go run ./cmd/uds -in examples/sales-overview.yaml -target metabase -out metabase-out
go run ./cmd/uds -in examples/sales-overview.yaml -target echarts -out echarts-out
go run ./cmd/uds -in examples/sales-overview-semantic.yaml \
  -models examples/ \
  -target cube -out orders.cube.yaml
```

| Flag | Meaning |
|------|---------|
| `-in` | Dashboard YAML. Required. |
| `-target` | `excalidraw` (default), `metabase`, `echarts`, or `cube`. |
| `-out` | File for Excalidraw and Cube. Directory for Metabase and ECharts. Defaults: `out.excalidraw`, `metabase-out`, `echarts-out`, `<model>.cube.yaml`. |
| `-models` | Comma-separated SemanticModel files or directories. A directory is scanned one level for `*.yaml` / `*.yml`. A dashboard sitting in that folder is skipped (`skipped <path> (kind "Dashboard")` on stderr). An explicit file that is not a SemanticModel still fails validation. |
| `-strict` | Intent/type mismatches fail the run. Otherwise they are warnings. |
| `-database` | Metabase. Replaces the database name on the base table and on every join. |
| `-schema`, `-table` | Metabase. Replace schema and table on the base table only. A joined model keeps its own schema and table. |

Cube reads the first model loaded by `-models`. The dashboard file is still required as `-in`.

## Metabase

Writes Representations YAML under the output directory:

- `collections/main/cards/<id>_<slug>.yaml` for each analytical widget
- `collections/main/dashboards/<id>_<slug>.yaml` for the dashboard

Each card is an MBQL query. Entity ids are a 21-character hash of the name (NanoID alphabet), so a re-run of the same spec does not churn ids.

The card table comes from the measure model's `spec.source`, then the dashboard `spec.source`, then `Sample Database` / `PUBLIC` / `ORDERS`. A model that needed one of those defaults prints a warning. `-database` fills every table on the card. `-schema` and `-table` fill the base table.

A second model is a left join when a measure, breakout, pivot field, fixed filter, or measure `where` names it. One side of the shared entity must be `primary`. If every model marks that entity `foreign`, the join fails until one model declares it `primary`. Two foreign keys fail the card. A dashboard filter does not add a join. If the filter's model is not already on the card, that dashcard leaves the parameter unmapped and warns.

`simple` and `ltm` metrics compile to the named measure (`ltm` as an `offset` window of the breakout grain). `ratio` compiles to numerator `/` denominator, with no zero-denominator guard. `derived` compiles to its expression. `cumulative` is skipped, with a warning. An unknown measure name is skipped, with a warning.

Series `type`, `axis`, and `color` are written as `series_settings`. A parameter with `swaps` is emitted as a control and does not change the card query; the emitter warns. Widget `actions` are not emitted, and the emitter warns.

`titleTemplate` on a tab or a widget substitutes `{{parameter}}` from the parameter `default` into the tab name and the card `name`. The raw template is also stored as the tab's `title_template`. Widget `parameters` only feed that substitution. An MBQL card has no template tags.

`text` and `image` become virtual text cards. An image card's body is `src`.

Operators in the YAML (`=`, `-`, `/`, and the other unsafe scalars) are double-quoted so a strict YAML reader accepts the file.

## Cube

One `cubes:` document from the first SemanticModel: dimensions (including `number`), measures, measure `expression` as `{measure}` SQL, measure `where` as Cube filters, measure `ltm` as `type: sum` plus `rolling_window: {trailing: "12 month", offset: end}`, and `ratio` metrics as calculated measures. `derived`, `cumulative`, and metric `ltm` are not emitted.

## ECharts

One TypeScript `export const` per widget. Series data is a placeholder. The option carries series `type`, `axis`, `color`, and `tooltipFields`, a raw `titleTemplate` when set, widget `parameters` (including `swaps`) as `udsParameters`, `timeGrain` on the category axis name, `goalLine` as `markLine`, and `text` / `image` as a small non-chart const.

## Excalidraw

One `.excalidraw` file. Rectangles are coloured by chart type. Labels show type, intent, the raw `titleTemplate` when set, `timeGrain`, fixed filters, series type and axis, and the measure title (SemanticModel title when `-models` loaded it). Tabs, filters (`+search` when `searchable`), and parameters are drawn above the grid.

## Not implemented

Lightdash and Grafana have no emitter. Widget `actions` (drill) are not emitted. See [docs/targets.md](../docs/targets.md) and [docs/cross-target.md](../docs/cross-target.md).
