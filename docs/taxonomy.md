# Chart and Dashboard Message Taxonomy

UDS treats the *message* a widget conveys as a first-class concept. Chart type is an encoding choice; intent is the semantic goal.

This keeps the language thin and LLM-friendly while making validation, multi-target compilation, and visual review more robust.

## Canonical intents

These cover Stephen Few’s seven quantitative relationships plus the practical groupings used by most BI and design systems.

| Intent | What it answers | Preferred encodings (current UDS types) | Notes |
|--------|-----------------|----------------------------------------|-------|
| `nominal_comparison` | How do discrete categories compare? | `bar` | Unordered categories. |
| `ranking` | What is the ordered magnitude? | `bar` | Prefer sorted descending (or ascending). |
| `time_series` | How did it change over time? | `line`, `area` | Temporal dimension on the horizontal axis. |
| `part_to_whole` | What share does each part contribute? | `stacked_bar`, `pie` | Prefer stacked bars for precise comparison; limit pies to few parts. |
| `deviation` | How do values differ from a reference? | `bar`, `line` | Use a reference line or baseline (see `options.goalLine`). |
| `distribution` | How are values spread? | `bar` (as histogram), `heatmap` | Binning or density. |
| `correlation` | Do two measures move together? | `scatter`, `heatmap` | Two quantitative measures. |
| `lookup` | Exact values or single KPIs | `table`, `number` | When precision or scanning matters more than pattern. |

Optional future extensions (not yet in the schema enum): `flow` (Sankey/funnel), `geographic`, `hierarchy` (treemap/sunburst).

## Validation

The CLI and parser run a lightweight check on load:

- JSON Schema validation (embedded `schema.json`) runs first — missing required fields and bad enums fail hard.
- Unknown `intent` values are **errors** (hard fail).
- A known intent paired with a non-preferred `type` produces a **warning** on stderr. Compilation still succeeds so existing specs remain valid.
- Missing intent is always fine (optional in v1).
- Pass `--strict` to promote those warnings to fatal errors.

Example warning:
```
warning: widget "deals_by_region" (Deals by Region): intent "ranking" prefers [bar], got type "pie"
```

## Visual review (Excalidraw)

Widget labels in the Excalidraw mock now show:
```
{type}
{intent}   ← only when present
{title}
```
Colour remains driven by chart type so the layout is still instantly scannable.

## Mapping guidance

| Intent | Required shape (rough) | Recommended options |
|--------|------------------------|---------------------|
| `ranking` | 1 measure + 1 categorical dimension | `sort: desc` (or asc) |
| `time_series` | 1+ measures + 1 temporal dimension | temporal grain, optional `goalLine` |
| `part_to_whole` | 1 measure + 1 categorical | stacked or percentage |
| `deviation` | 1 measure (+ reference) | reference line / target |
| `correlation` | 2 measures | trend line optional |
| `lookup` | any | exact values, no aggregation needed for tables |

## Target notes (lossiness)

- **Metabase**:
  - `ranking` → `order-by` descending on the first aggregation.
  - `part_to_whole` (or `stacked_bar`) → `graph.stacking: stacked`.
  - `deviation` / `options.goalLine` → `graph.goal_value`.
- **Lightdash / Grafana**: Most intents map cleanly. Ranking requires the target to honour sort order. Pie charts for part-to-whole are lossy for precise comparison on every target.
- **ECharts**: Full control; markLine for deviation, sorting is client-side.
- **Excalidraw**: Intent appears in the widget label. Colour is still by chart type.

## Sources

- Stephen Few, “Selecting the Right Graph for Your Message” (Perceptual Edge) — seven quantitative relationship types.
- Common practical groupings: Comparison / Trend / Composition / Distribution / Relationship (used across design systems and BI tools).
- Cleveland & McGill perceptual accuracy hierarchy (position > length > angle > area) underpins why certain encodings are preferred for each intent.
- Shneiderman’s task-by-data-type taxonomy informs interaction patterns (overview → zoom/filter → details) more than chart choice itself.

The `intent` field on widgets is optional in v1 so existing specs remain valid. New specs and LLM authors should supply it.
