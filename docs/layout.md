# Layout

UDS uses a **24-column grid**.

- `x` and `w` are column units (0–23 for x, 1–24 for w).
- `y` and `h` are row units.
- One row unit is approximately **30px** in Metabase and most other targets. Exact pixel height is target-dependent; the unit is the contract, not the pixel value.

```yaml
layout: { x: 0, y: 0, w: 16, h: 8 }  # 16 columns wide, 8 rows tall (~240px)
```

Widgets must not overlap. Tabs are independent grids.

Metabase copies `x`, `y`, `w`, `h` onto the dashcard as `col`, `row`, `size_x`, `size_y`. Excalidraw uses 32px per column and 36px per row. ECharts does not draw a grid.
