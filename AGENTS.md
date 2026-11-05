# Authoring UDS (for LLMs)

You are writing a Universal Dashboard Specification (UDS) YAML file.

## Rules

1. **Always set `intent`** on every analytical widget. Omit it on `text` and `image`. Use one of:
   - `nominal_comparison` — discrete categories, no natural order → prefer `bar` (`combo` when series mix bar and line)
   - `ranking` — ordered magnitude → prefer `bar` (mention sort in options if needed)
   - `time_series` — change over time → prefer `line` or `area` (`combo` when series mix bar and line)
   - `part_to_whole` — share of a total → prefer `stacked_bar` (limit `pie`)
   - `deviation` — difference from a reference → prefer `bar` or `line` (`combo` is accepted) + `options.goalLine`
   - `distribution` — spread of values → prefer `bar` or `heatmap`
   - `correlation` — relationship between two measures → prefer `scatter` or `heatmap`
   - `lookup` — exact values or single KPI → prefer `table` or `number`

2. **Match preferred type** for the intent. If you choose a non-preferred type, the CLI will warn.

3. **Layout** uses a 24-column grid: `layout: { x, y, w, h }`.

4. **Source** (optional but recommended for Metabase). A card that references a SemanticModel uses that model's `source`. A dimension from a second model is a left join on a shared entity name. Inline cards use the dashboard `source`:
   ```yaml
   spec:
     source:
       database: My Database
       schema: public
       table: orders
   ```

5. **Measures and dimensions** are declared once at the top and referenced by name in widgets. Prefer a separate `kind: SemanticModel` (see `docs/semantic-model.md`) when the same metrics will be reused across dashboards.

6. Keep the file small and regular. Prefer the example in `examples/sales-overview.yaml` as a template.

## Validation

The CLI validates against the JSON Schema on load and checks intent/type pairings.
Unknown intents fail. Type mismatches warn (or fail with `--strict`).

See `docs/taxonomy.md` for the full rationale.
