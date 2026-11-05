# Cross-target notes

| Concept              | Metabase          | Lightdash     | Grafana v2     | ECharts     | Excalidraw |
|----------------------|-------------------|---------------|----------------|-------------|------------|
| Temporal grain       | field + unit      | timeInterval  | interval       | custom      | label only |
| Goal line            | viz settings      | reference line| threshold      | markLine    | annotation |
| Tabs                 | native            | native        | TabsLayout     | separate    | groups     |
| Filter wiring        | parameter mapping | dashboard filter | variable     | external    | arrows     |
| Stable IDs           | NanoID manifest   | file names    | panel ids      | n/a         | element ids|
| Measure `where`      | `*-where` agg     | query filter  | SQL `WHERE`    | external    | not shown  |
| Measure expression   | MBQL arithmetic   | SQL expr      | SQL expr       | external    | not shown  |
| Widget `timeGrain`   | `temporal-unit`   | timeInterval  | interval       | axis name   | label      |
| Tab `titleTemplate`  | not emitted       | title         | title          | n/a         | label      |
| `text` / `image`     | virtual text card | markdown tile | text panel     | const       | rectangle  |
| Pivot row mix        | order stored; measures in values | lossy | lossy | n/a | not shown |

Temporal grain encoding is the single hardest portable concept.
