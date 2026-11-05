package parser

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
//go:embed semantic-model.json
var schemaFS embed.FS

var dashboardSchema *jsonschema.Schema
var semanticSchema *jsonschema.Schema

func init() {
	db, err := schemaFS.ReadFile("schema.json")
	if err != nil {
		panic(err)
	}
	sb, err := schemaFS.ReadFile("semantic-model.json")
	if err != nil {
		panic(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("dashboard.json", bytes.NewReader(db)); err != nil {
		panic(err)
	}
	if err := c.AddResource("semantic-model.json", bytes.NewReader(sb)); err != nil {
		panic(err)
	}
	dashboardSchema, err = c.Compile("dashboard.json")
	if err != nil {
		panic(err)
	}
	semanticSchema, err = c.Compile("semantic-model.json")
	if err != nil {
		panic(err)
	}
}

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
	Source     *Source     `yaml:"source,omitempty"`
	Models     []string    `yaml:"models,omitempty"`
	Measures   []Measure   `yaml:"measures"`
	Dimensions []Dimension `yaml:"dimensions"`
	Filters    []Filter    `yaml:"filters,omitempty"`
	Tabs       []Tab       `yaml:"tabs,omitempty"`
	Widgets    []Widget    `yaml:"widgets"`
}

type Source struct {
	Database string `yaml:"database,omitempty"`
	Schema   string `yaml:"schema,omitempty"`
	Table    string `yaml:"table,omitempty"`
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

// SemanticModel is a reusable metrics / semantic layer definition.
type SemanticModel struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   Metadata          `yaml:"metadata"`
	Spec       SemanticModelSpec `yaml:"spec"`
}

type SemanticModelSpec struct {
	Source     *Source    `yaml:"source,omitempty"`
	Entities   []Entity   `yaml:"entities,omitempty"`
	Dimensions []Dimension `yaml:"dimensions"`
	Measures   []Measure  `yaml:"measures"`
	Metrics    []Metric   `yaml:"metrics,omitempty"`
}

type Entity struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"` // primary | foreign
	Field       string `yaml:"field"`
	Description string `yaml:"description,omitempty"`
}

type Metric struct {
	Name        string `yaml:"name"`
	Title       string `yaml:"title,omitempty"`
	Type        string `yaml:"type"` // simple | ratio | cumulative | derived
	Description string `yaml:"description,omitempty"`
	Measure     string `yaml:"measure,omitempty"`
	Numerator   string `yaml:"numerator,omitempty"`
	Denominator string `yaml:"denominator,omitempty"`
	InputMetric string `yaml:"input_metric,omitempty"`
	GrainToDate string `yaml:"grain_to_date,omitempty"`
	Expression  string `yaml:"expression,omitempty"`
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

	// Validate against JSON Schema (YAML → generic map → JSON)
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	// jsonschema expects map[string]any, not map[any]any
	normalized, err := normalize(raw)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	var doc any
	if err := json.Unmarshal(jb, &doc); err != nil {
		return nil, err
	}
	if err := dashboardSchema.Validate(doc); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	var d Dashboard
	if err := yaml.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// LoadSemanticModel loads and validates a SemanticModel YAML file.
func LoadSemanticModel(path string) (*SemanticModel, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	normalized, err := normalize(raw)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	var doc any
	if err := json.Unmarshal(jb, &doc); err != nil {
		return nil, err
	}
	if err := semanticSchema.Validate(doc); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	var m SemanticModel
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Registry holds loaded SemanticModels for name resolution.
type Registry struct {
	models map[string]*SemanticModel
}

func NewRegistry() *Registry {
	return &Registry{models: make(map[string]*SemanticModel)}
}

func (r *Registry) Add(m *SemanticModel) {
	r.models[m.Metadata.Name] = m
}

func (r *Registry) Get(name string) (*SemanticModel, bool) {
	m, ok := r.models[name]
	return m, ok
}

// Models returns all loaded SemanticModels (order is not guaranteed).
func (r *Registry) Models() []*SemanticModel {
	out := make([]*SemanticModel, 0, len(r.models))
	for _, m := range r.models {
		out = append(out, m)
	}
	return out
}

// ResolveMeasure looks up a measure or metric by qualified or unqualified name.
// Returns the model name (if qualified), the measure/metric name, and whether it was found.
func (r *Registry) ResolveMeasure(ref string, dash *Dashboard) (modelName, name string, found bool) {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) == 2 {
		modelName, name = parts[0], parts[1]
		if m, ok := r.models[modelName]; ok {
			for _, meas := range m.Spec.Measures {
				if meas.Name == name {
					return modelName, name, true
				}
			}
			for _, met := range m.Spec.Metrics {
				if met.Name == name {
					return modelName, name, true
				}
			}
		}
		return modelName, name, false
	}
	// unqualified — look in dashboard first
	name = ref
	for _, meas := range dash.Spec.Measures {
		if meas.Name == name {
			return "", name, true
		}
	}
	return "", name, false
}

// normalize converts map[any]any (from yaml) into map[string]any for json.
func normalize(v any) (any, error) {
	switch t := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string map key %v", k)
			}
			nv, err := normalize(val)
			if err != nil {
				return nil, err
			}
			m[ks] = nv
		}
		return m, nil
	case []any:
		for i, val := range t {
			nv, err := normalize(val)
			if err != nil {
				return nil, err
			}
			t[i] = nv
		}
		return t, nil
	default:
		return v, nil
	}
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

// SourceOrDefault returns the source from the spec, or the provided defaults
// when the spec does not declare one.
func (d *Dashboard) SourceOrDefault(db, schema, table string) (string, string, string) {
	if d.Spec.Source != nil {
		if d.Spec.Source.Database != "" {
			db = d.Spec.Source.Database
		}
		if d.Spec.Source.Schema != "" {
			schema = d.Spec.Source.Schema
		}
		if d.Spec.Source.Table != "" {
			table = d.Spec.Source.Table
		}
	}
	return db, schema, table
}
