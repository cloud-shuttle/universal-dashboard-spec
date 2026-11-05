package parser

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Dashboard struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
}

type Spec struct {
	Measures   []Measure   `yaml:"measures"`
	Dimensions []Dimension `yaml:"dimensions"`
	Filters    []Filter    `yaml:"filters,omitempty"`
	Tabs       []Tab       `yaml:"tabs,omitempty"`
	Widgets    []Widget    `yaml:"widgets"`
}

type Measure struct {
	Name        string `yaml:"name"`
	Title       string `yaml:"title,omitempty"`
	Aggregation string `yaml:"aggregation"`
	Field       string `yaml:"field,omitempty"`
	Format      string `yaml:"format,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Dimension struct {
	Name          string `yaml:"name"`
	Title         string `yaml:"title,omitempty"`
	Field         string `yaml:"field,omitempty"`
	Type          string `yaml:"type,omitempty"`
	TemporalGrain string `yaml:"temporalGrain,omitempty"`
}

type Filter struct {
	Name      string `yaml:"name"`
	Title     string `yaml:"title,omitempty"`
	Dimension string `yaml:"dimension"`
	Required  bool   `yaml:"required,omitempty"`
}

type Tab struct {
	Name  string `yaml:"name"`
	Title string `yaml:"title"`
}

type Widget struct {
	ID         string         `yaml:"id"`
	Type       string         `yaml:"type"`
	Intent     string         `yaml:"intent,omitempty"`
	Title      string         `yaml:"title"`
	Tab        string         `yaml:"tab,omitempty"`
	Measures   []string       `yaml:"measures,omitempty"`
	Dimensions []string       `yaml:"dimensions,omitempty"`
	Filters    []string       `yaml:"filters,omitempty"`
	Layout     Layout         `yaml:"layout"`
	Options    map[string]any `yaml:"options,omitempty"`
}

type Layout struct {
	X int `yaml:"x"`
	Y int `yaml:"y"`
	W int `yaml:"w"`
	H int `yaml:"h"`
}

// knownIntents is the canonical set from docs/taxonomy.md
var knownIntents = map[string]bool{
	"nominal_comparison": true,
	"ranking":            true,
	"time_series":        true,
	"part_to_whole":      true,
	"deviation":          true,
	"distribution":       true,
	"correlation":        true,
	"lookup":             true,
}

// preferredTypes maps intent → preferred chart types. Empty means any is acceptable.
var preferredTypes = map[string][]string{
	"nominal_comparison": {"bar"},
	"ranking":            {"bar"},
	"time_series":        {"line", "area"},
	"part_to_whole":      {"stacked_bar", "pie"},
	"deviation":          {"bar", "line"},
	"distribution":       {"bar", "heatmap"},
	"correlation":        {"scatter", "heatmap"},
	"lookup":             {"table", "number"},
}

func Load(path string) (*Dashboard, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d Dashboard
	if err := yaml.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Validate checks known intents and recommended type pairings.
// Returns a list of warning messages (non-fatal) and an error only for hard failures
// (unknown intent). Type mismatches are warnings so existing specs keep working.
func (d *Dashboard) Validate() (warnings []string, err error) {
	for i, w := range d.Spec.Widgets {
		if w.Intent == "" {
			continue
		}
		if !knownIntents[w.Intent] {
			return nil, fmt.Errorf("widget %q (%s): unknown intent %q", w.ID, w.Title, w.Intent)
		}
		preferred, ok := preferredTypes[w.Intent]
		if !ok {
			continue
		}
		found := false
		for _, p := range preferred {
			if w.Type == p {
				found = true
				break
			}
		}
		if !found {
			warnings = append(warnings, fmt.Sprintf(
				"widget %q (%s): intent %q prefers %v, got type %q",
				w.ID, w.Title, w.Intent, preferred, w.Type,
			))
		}
		_ = i // silence unused in case of future use
	}
	return warnings, nil
}
