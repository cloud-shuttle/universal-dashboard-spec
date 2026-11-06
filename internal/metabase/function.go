package metabase

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

// nativeSource is the MBQL source-query for a table function.
// Arguments become template tags so dashboard controls change the call.
func nativeSource(src cardSource, d *parser.Dashboard) map[string]any {
	return map[string]any{
		"native": map[string]any{
			"query":         functionSQL(src),
			"template-tags": templateTags(src, d),
		},
	}
}

func functionSQL(src cardSource) string {
	fn := src.function
	if fn != "" && !strings.Contains(fn, ".") && src.schema != "" {
		fn = src.schema + "." + fn
	}
	parts := make([]string, 0, len(src.arguments))
	for _, arg := range src.arguments {
		switch {
		case arg.Parameter == "" && arg.Filter == "":
			parts = append(parts, sqlLiteral(arg.Default))
		case len(arg.Map) > 0:
			parts = append(parts, caseArg(arg.Name, arg.Map))
		default:
			parts = append(parts, "{{"+arg.Name+"}}")
		}
	}
	// ponytail: a multi-select filter is one text tag; Metabase does not expand it into a SQL list inside the call. Change the call when a function takes an array.
	return "SELECT * FROM " + fn + "(" + strings.Join(parts, ", ") + ")"
}

func caseArg(name string, values map[string]any) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("CASE {{")
	b.WriteString(name)
	b.WriteString("}}")
	for _, k := range keys {
		b.WriteString(" WHEN ")
		b.WriteString(sqlLiteral(k))
		b.WriteString(" THEN ")
		b.WriteString(sqlLiteral(values[k]))
	}
	b.WriteString(" END")
	return b.String()
}

func sqlLiteral(v any) string {
	switch n := v.(type) {
	case int:
		return strconv.Itoa(n)
	case int64:
		return strconv.FormatInt(n, 10)
	case float64:
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	case string:
		return "'" + strings.ReplaceAll(n, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(n), "'", "''") + "'"
	}
}

func templateTags(src cardSource, d *parser.Dashboard) map[string]any {
	tags := map[string]any{}
	for _, arg := range src.arguments {
		if arg.Parameter == "" && arg.Filter == "" {
			continue
		}
		tag := map[string]any{
			"id":           stableID("tag-" + src.base + "-" + arg.Name),
			"name":         arg.Name,
			"display-name": argDisplay(d, arg),
			"type":         argType(d, arg),
		}
		if def, ok := tagDefault(d, arg); ok {
			tag["default"] = def
		}
		tags[arg.Name] = tag
	}
	return tags
}

func tagDefault(d *parser.Dashboard, arg parser.SourceArgument) (any, bool) {
	if arg.Default != nil {
		return arg.Default, true
	}
	if arg.Parameter != "" {
		for _, p := range d.Spec.Parameters {
			if p.Name == arg.Parameter && p.Default != nil {
				return p.Default, true
			}
		}
	}
	return nil, false
}

func argDisplay(d *parser.Dashboard, arg parser.SourceArgument) string {
	if arg.Parameter != "" {
		for _, p := range d.Spec.Parameters {
			if p.Name == arg.Parameter && p.Title != "" {
				return p.Title
			}
		}
		return arg.Parameter
	}
	if f := findFilter(d, arg.Filter); f != nil && f.Title != "" {
		return f.Title
	}
	if arg.Filter != "" {
		return arg.Filter
	}
	return arg.Name
}

func argType(d *parser.Dashboard, arg parser.SourceArgument) string {
	if arg.Parameter != "" {
		for _, p := range d.Spec.Parameters {
			if p.Name == arg.Parameter {
				switch p.Type {
				case "number":
					return "number"
				case "date", "date_range":
					return "date"
				}
			}
		}
	}
	return "text"
}

func (s cardSource) argumentForFilter(name string) (parser.SourceArgument, bool) {
	for _, arg := range s.arguments {
		if arg.Filter == name {
			return arg, true
		}
	}
	return parser.SourceArgument{}, false
}

func (s cardSource) argumentForParameter(name string) (parser.SourceArgument, bool) {
	for _, arg := range s.arguments {
		if arg.Parameter == name {
			return arg, true
		}
	}
	return parser.SourceArgument{}, false
}
