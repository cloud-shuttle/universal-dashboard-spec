# Cross-target notes

Metabase, ECharts, Excalidraw, and Cube have emitters. The Lightdash and Grafana columns are the intended mapping. Those two emitters are not in this repo. Cube is a SemanticModel target, so dashboard-only rows do not apply to it. See [semantic-model.md](semantic-model.md) for which metric types Cube writes.

| Concept              | Metabase          | Lightdash     | Grafana v2     | ECharts     | Excalidraw |
|----------------------|-------------------|---------------|----------------|-------------|------------|
| Temporal grain       | field + unit      | timeInterval  | interval       | custom      | label only |
| Goal line            | viz settings      | reference line| threshold      | markLine    | not shown  |
| Tabs                 | native            | native        | TabsLayout     | not emitted | tab headers |
| Filter wiring        | field target; unmapped when that model is not in the query | dashboard filter | variable     | external    | label, no arrows |
| Stable IDs           | hash, NanoID alphabet | file names | panel ids   | n/a         | element ids|
| Measure `where`      | `*-where` agg     | query filter  | SQL `WHERE`    | external    | not shown  |
| Measure expression   | MBQL arithmetic   | SQL expr      | SQL expr       | external    | not shown  |
| Widget `timeGrain`   | `temporal-unit`   | timeInterval  | interval       | axis name   | label      |
| Tab `titleTemplate`  | tab and card name, default substituted; raw template kept on the tab | title | title   | raw title   | raw label  |
| Model source         | model table, else dashboard source, else Sample Database / PUBLIC / ORDERS with a warning. `-database` sets every table | query filter | SQL join | external | not shown |
| Model join           | left join only when a query field needs it; one side must be `primary`; a filter does not join | not implemented | not implemented | external | not shown |
| `ratio` / `derived`  | MBQL expression   | SQL expr      | SQL expr       | external    | title if set |
| `cumulative`         | skipped, warning  | running total | running total  | external    | not shown |
| LTM                  | `offset` window   | rolling window| SQL window     | external    | not shown  |
| `text` / `image`     | virtual text card | markdown tile | text panel     | const       | rectangle  |
| Pivot row mix        | column split; measures stay listed | lossy | lossy | n/a | not shown |

Temporal grain encoding is the single hardest portable concept.
