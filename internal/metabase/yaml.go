package metabase

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// marshalDoc encodes v as YAML, quoting scalars that a strict reader rejects
// when they are plain. MBQL operators such as "=" and "-" are the usual ones.
func marshalDoc(v any) ([]byte, error) {
	var doc yaml.Node
	if err := doc.Encode(v); err != nil {
		return nil, err
	}
	quoteUnsafe(&doc)
	return yaml.Marshal(&doc)
}

func quoteUnsafe(n *yaml.Node) {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" && unsafePlain(n.Value) {
		n.Style = yaml.DoubleQuotedStyle
	}
	for _, c := range n.Content {
		quoteUnsafe(c)
	}
}

func unsafePlain(s string) bool {
	switch s {
	case "", "=", "==", "!=", "<", ">", "<=", ">=", "+", "-", "*", "/",
		"yes", "no", "on", "off", "y", "n", "null", "~", "true", "false":
		return true
	}
	if strings.ContainsAny(s, ":{}[],&*!|>'\"%@`#") || strings.Contains(s, "{{") {
		return true
	}
	switch s[0] {
	case ' ', '-', '?', '=', '+', '&', '*', '!':
		return true
	}
	return false
}
