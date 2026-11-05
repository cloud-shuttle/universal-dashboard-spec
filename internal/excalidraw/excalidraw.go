package excalidraw

import (
	"encoding/json"

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

func Emit(d *parser.Dashboard) ([]byte, error) {
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
			Label:           &Label{Text: filter.Title},
		})
		x += 160
	}

	// Widgets (simple absolute layout from 24-col grid)
	const colW = 32
	const rowH = 36
	for _, w := range d.Spec.Widgets {
		color := chartColor(w.Type)
		f.Elements = append(f.Elements, Element{
			ID:              "widget-" + w.ID,
			Type:            "rectangle",
			X:               40 + w.Layout.X*colW,
			Y:               160 + w.Layout.Y*rowH,
			Width:           w.Layout.W * colW,
			Height:          w.Layout.H * rowH,
			StrokeColor:     color,
			BackgroundColor: lighten(color),
			FillStyle:       "solid",
			Label:           &Label{Text: w.Type + "\n" + w.Title},
		})
	}

	return json.MarshalIndent(f, "", "  ")
}

func chartColor(t string) string {
	switch t {
	case "line":
		return "#2f9e44"
	case "bar":
		return "#e8590c"
	case "stacked_bar":
		return "#1971c2"
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
	default:
		return "#f8f9fa"
	}
}
