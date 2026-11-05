# Cross-target notes

| Concept              | Metabase          | Lightdash     | Grafana v2     | ECharts     | Excalidraw |
|----------------------|-------------------|---------------|----------------|-------------|------------|
| Temporal grain       | field + unit      | timeInterval  | interval       | custom      | label only |
| Goal line            | viz settings      | reference line| threshold      | markLine    | annotation |
| Tabs                 | native            | native        | TabsLayout     | separate    | groups     |
| Filter wiring        | parameter mapping | dashboard filter | variable     | external    | arrows     |
| Stable IDs           | NanoID manifest   | file names    | panel ids      | n/a         | element ids|

Temporal grain encoding is the single hardest portable concept.
