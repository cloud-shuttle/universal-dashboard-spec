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
	Parameters []Parameter `yaml:"parameters,omitempty"`
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

type Parameter struct {
	Name        string            `yaml:"name"`
	Title       string            `yaml:"title,omitempty"`
	Type        string            `yaml:"type"`
	Values      []string          `yaml:"values,omitempty"`
	Affects     string            `yaml:"affects,omitempty"`
	Swaps       map[string]string `yaml:"swaps,omitempty"`
	Default     any               `yaml:"default,omitempty"`
	Required    bool              `yaml:"required,omitempty"`
	Description string            `yaml:"description,omitempty"`
}

// Predicate is a constant row comparison. Used by measure where clauses and widget fixedFilters.
type Predicate struct {
	Dimension string `yaml:"dimension"`
	Operator  string `yaml:"operator"`
	Value     any    `yaml:"value,omitempty"`
}

// PivotField is one entry in a pivot row or column order.
// Exactly one of Dimension or Measure is set.
type PivotField struct {
	Dimension string `yaml:"dimension,omitempty"`
	Measure   string `yaml:"measure,omitempty"`
}

type Measure struct {
	Name        string      `yaml:"name"`
	Title       string      `yaml:"title,omitempty"`
	Aggregation string      `yaml:"aggregation,omitempty"`
	Expression  string      `yaml:"expression,omitempty"`
	Field       string      `yaml:"field,omitempty"`
	Format      string      `yaml:"format,omitempty"`
	Description string      `yaml:"description,omitempty"`
	Where       []Predicate `yaml:"where,omitempty"`
}

type Dimension struct {
	Name          string   `yaml:"name"`
	Title         string   `yaml:"title,omitempty"`
	Field         string   `yaml:"field,omitempty"`
	Type          string   `yaml:"type,omitempty"`
	TemporalGrain string   `yaml:"temporalGrain,omitempty"`
	Grains        []string `yaml:"grains,omitempty"`
}

type Filter struct {
	Name        string   `yaml:"name"`
	Title       string   `yaml:"title,omitempty"`
	Dimension   string   `yaml:"dimension,omitempty"`
	Bindings    []string `yaml:"bindings,omitempty"`
	Searchable  bool     `yaml:"searchable,omitempty"`
	Control     string   `yaml:"control,omitempty"`
	Values      []string `yaml:"values,omitempty"`
	Multi       bool     `yaml:"multi,omitempty"`
	CascadeFrom string   `yaml:"cascadeFrom,omitempty"`
	Required    bool     `yaml:"required,omitempty"`
}

type Tab struct {
	Name          string `yaml:"name"`
	Title         string `yaml:"title"`
	TitleTemplate string `yaml:"titleTemplate,omitempty"`
}

type Widget struct {
	ID            string         `yaml:"id"`
	Type          string         `yaml:"type"`
	Intent        string         `yaml:"intent,omitempty"`
	Title         string         `yaml:"title"`
	TitleTemplate string         `yaml:"titleTemplate,omitempty"`
	Tab           string         `yaml:"tab,omitempty"`
	Measures      []string       `yaml:"measures,omitempty"`
	Dimensions    []string       `yaml:"dimensions,omitempty"`
	Filters       []string       `yaml:"filters,omitempty"`
	TimeGrain     string         `yaml:"timeGrain,omitempty"`
	Content       string         `yaml:"content,omitempty"`
	Src           string         `yaml:"src,omitempty"`
	FixedFilters  []Predicate    `yaml:"fixedFilters,omitempty"`
	Rows          []PivotField   `yaml:"rows,omitempty"`
	Columns       []PivotField   `yaml:"columns,omitempty"`
	Parameters    []string       `yaml:"parameters,omitempty"`
	Actions       []Action       `yaml:"actions,omitempty"`
	Series        []Series       `yaml:"series,omitempty"`
	Layout        Layout         `yaml:"layout"`
	Options       map[string]any `yaml:"options,omitempty"`
}

type Series struct {
	Name          string   `yaml:"name"`
	Type          string   `yaml:"type,omitempty"`
	Axis          string   `yaml:"axis,omitempty"`
	Color         string   `yaml:"color,omitempty"`
	TooltipFields []string `yaml:"tooltipFields,omitempty"`
}

type Action struct {
	Type              string            `yaml:"type"`
	Target            string            `yaml:"target,omitempty"`
	ParameterMappings map[string]string `yaml:"parameterMappings,omitempty"`
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
	Source     *Source     `yaml:"source,omitempty"`
	Entities   []Entity    `yaml:"entities,omitempty"`
	Dimensions []Dimension `yaml:"dimensions"`
	Measures   []Measure   `yaml:"measures"`
	Metrics    []Metric    `yaml:"metrics,omitempty"`
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
	Type        string `yaml:"type"` // simple | ratio | cumulative | derived | ltm
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
	"nominal_comparison": {"bar", "combo"},
	"ranking":            {"bar"},
	"time_series":        {"line", "area", "combo"},
	"part_to_whole":      {"stacked_bar", "pie"},
	"deviation":          {"bar", "line", "diverging", "combo"},
	"distribution":       {"bar", "heatmap"},
	"correlation":        {"scatter", "heatmap"},
	"lookup":             {"table", "number", "pivot"},
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

// Validate checks measures, filters, pivot entries, and intent/type pairings.
// Unknown intent, a bad expression, and a measure with both aggregation and
// expression are errors. A known intent paired with a non-preferred type is a warning.
// text and image widgets have no analytical intent, so that check is skipped.
func (d *Dashboard) Validate() (warnings []string, err error) {
	if err := validateMeasures(d.Spec.Measures); err != nil {
		return nil, err
	}
	for _, f := range d.Spec.Filters {
		if f.Dimension == "" && len(f.Bindings) == 0 {
			return nil, fmt.Errorf("filter %q: set dimension or bindings", f.Name)
		}
	}
	for _, w := range d.Spec.Widgets {
		if err := validatePredicates(w.FixedFilters); err != nil {
			return nil, fmt.Errorf("widget %q: %w", w.ID, err)
		}
		if err := validatePivot(w.Rows); err != nil {
			return nil, fmt.Errorf("widget %q rows: %w", w.ID, err)
		}
		if err := validatePivot(w.Columns); err != nil {
			return nil, fmt.Errorf("widget %q columns: %w", w.ID, err)
		}
		switch w.Type {
		case "text":
			if w.Content == "" {
				return nil, fmt.Errorf("widget %q: text requires content", w.ID)
			}
			continue
		case "image":
			if w.Src == "" {
				return nil, fmt.Errorf("widget %q: image requires src", w.ID)
			}
			continue
		}
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
	}
	return warnings, nil
}

// Validate checks semantic measures: aggregation xor expression, expression grammar, and where clauses.
func (m *SemanticModel) Validate() error {
	return validateMeasures(m.Spec.Measures)
}

func validateMeasures(measures []Measure) error {
	for _, m := range measures {
		hasAgg := m.Aggregation != ""
		hasExpr := m.Expression != ""
		if hasAgg == hasExpr {
			return fmt.Errorf("measure %q: set aggregation or expression, not both", m.Name)
		}
		if hasExpr {
			if _, err := ParseExpression(m.Expression); err != nil {
				return fmt.Errorf("measure %q: %w", m.Name, err)
			}
		}
		if err := validatePredicates(m.Where); err != nil {
			return fmt.Errorf("measure %q: %w", m.Name, err)
		}
	}
	return nil
}

var predicateOps = map[string]bool{
	"eq": true, "neq": true, "in": true, "not_in": true,
	"gt": true, "gte": true, "lt": true, "lte": true,
}

func validatePredicates(ps []Predicate) error {
	for _, p := range ps {
		if p.Dimension == "" {
			return fmt.Errorf("predicate missing dimension")
		}
		if !predicateOps[p.Operator] {
			return fmt.Errorf("predicate on %q: unknown operator %q", p.Dimension, p.Operator)
		}
	}
	return nil
}

func validatePivot(items []PivotField) error {
	for _, item := range items {
		if (item.Dimension == "") == (item.Measure == "") {
			return fmt.Errorf("each entry needs a dimension or a measure")
		}
	}
	return nil
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
