package excalidraw

import (
	"encoding/json"
	"strings"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

type File struct {
	Type     string    `json:"type"`
	Version  int       `json:"version"`
	Source   string    `json:"source"`
	Elements []Element `json:"elements"`
	AppState struct {
		ViewBackgroundColor string `json:"viewBackgroundColor"`
	} `json:"appState"`
	Files map[string]any `json:"files"`
}

type Element struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	StrokeColor     string `json:"strokeColor"`
	BackgroundColor string `json:"backgroundColor"`
	FillStyle       string `json:"fillStyle"`
	Label           *Label `json:"label,omitempty"`
}

type Label struct {
	Text string `json:"text"`
}

func Emit(d *parser.Dashboard, reg *parser.Registry) ([]byte, error) {
	f := File{
		Type:    "excalidraw",
		Version: 2,
		Source:  "uds-transpiler",
		Files:   map[string]any{},
	}
	f.AppState.ViewBackgroundColor = "#ffffff"

	// Tabs
	x := 40
	for _, tab := range d.Spec.Tabs {
		f.Elements = append(f.Elements, Element{
			ID:              "tab-" + tab.Name,
			Type:            "rectangle",
			X:               x,
			Y:               40,
			Width:           120,
			Height:          40,
			StrokeColor:     "#1e1e1e",
			BackgroundColor: "#e7f5ff",
			FillStyle:       "solid",
			Label:           &Label{Text: tab.Title},
		})
		x += 140
	}

	// Filters
	x = 40
	for _, filter := range d.Spec.Filters {
		label := filter.Title
		if filter.Control != "" {
			label = filter.Control + ": " + label
		}
		f.Elements = append(f.Elements, Element{
			ID:              "filter-" + filter.Name,
			Type:            "rectangle",
			X:               x,
			Y:               100,
			Width:           140,
			Height:          32,
			StrokeColor:     "#1971c2",
			BackgroundColor: "#e7f5ff",
			FillStyle:       "solid",
			Label:           &Label{Text: label},
		})
		x += 160
	}

	// Parameters
	x = 40
	for _, p := range d.Spec.Parameters {
		label := p.Title
		if label == "" {
			label = p.Name
		}
		if p.Affects != "" {
			label = label + " (" + p.Affects + ")"
		}
		f.Elements = append(f.Elements, Element{
			ID:              "param-" + p.Name,
			Type:            "rectangle",
			X:               x,
			Y:               140,
			Width:           140,
			Height:          32,
			StrokeColor:     "#9c36b5",
			BackgroundColor: "#f3d9fa",
			FillStyle:       "solid",
			Label:           &Label{Text: label},
		})
		x += 160
	}

	// Widgets (simple absolute layout from 24-col grid)
	const colW = 32
	const rowH = 36
	for _, w := range d.Spec.Widgets {
		color := chartColor(w.Type)
		title := w.Title
		if w.TitleTemplate != "" {
			title = w.TitleTemplate
		}
		label := w.Type
		if w.Intent != "" {
			label = w.Type + "\n" + w.Intent
		}
		label = label + "\n" + title
		if titles := resolveMeasureTitles(w.Measures, d, reg); titles != "" {
			label = label + "\n" + titles
		}
		if len(w.Series) > 0 {
			var bits []string
			for _, s := range w.Series {
				bit := s.Name
				if s.Type != "" {
					bit = s.Type + ":" + bit
				}
				if s.Axis != "" {
					bit = bit + "(" + s.Axis + ")"
				}
				bits = append(bits, bit)
			}
			label = label + "\n" + strings.Join(bits, ", ")
		}
		f.Elements = append(f.Elements, Element{
			ID:              "widget-" + w.ID,
			Type:            "rectangle",
			X:               40 + w.Layout.X*colW,
			Y:               200 + w.Layout.Y*rowH,
			Width:           w.Layout.W * colW,
			Height:          w.Layout.H * rowH,
			StrokeColor:     color,
			BackgroundColor: lighten(color),
			FillStyle:       "solid",
			Label:           &Label{Text: strings.TrimSpace(label)},
		})
	}

	return json.MarshalIndent(f, "", "  ")
}

// resolveMeasureTitles returns a short comma-separated list of measure titles
// for the widget, preferring SemanticModel titles when a registry is present.
func resolveMeasureTitles(names []string, d *parser.Dashboard, reg *parser.Registry) string {
	if len(names) == 0 {
		return ""
	}
	var titles []string
	for _, n := range names {
		title := n
		if reg != nil {
			parts := strings.SplitN(n, ".", 2)
			if len(parts) == 2 {
				if m, ok := reg.Get(parts[0]); ok {
					for _, meas := range m.Spec.Measures {
						if meas.Name == parts[1] {
							if meas.Title != "" {
								title = meas.Title
							} else {
								title = meas.Name
							}
							break
						}
					}
					for _, met := range m.Spec.Metrics {
						if met.Name == parts[1] {
							if met.Title != "" {
								title = met.Title
							} else {
								title = met.Name
							}
							break
						}
					}
				}
			}
		} else {
			for _, meas := range d.Spec.Measures {
				if meas.Name == n && meas.Title != "" {
					title = meas.Title
					break
				}
			}
		}
		titles = append(titles, title)
	}
	return strings.Join(titles, ", ")
}

func chartColor(t string) string {
	switch t {
	case "line":
		return "#2f9e44"
	case "bar":
		return "#e8590c"
	case "stacked_bar":
		return "#1971c2"
	case "combo":
		return "#9c36b5"
	default:
		return "#495057"
	}
}

func lighten(c string) string {
	switch c {
	case "#2f9e44":
		return "#ebfbee"
	case "#e8590c":
		return "#fff4e6"
	case "#1971c2":
		return "#e7f5ff"
	case "#9c36b5":
		return "#f3d9fa"
	default:
		return "#f8f9fa"
	}
}
