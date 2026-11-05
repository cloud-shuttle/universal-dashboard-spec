package cube

import (
	"strings"
	"testing"

	"github.com/cloud-shuttle/universal-dashboard-spec/internal/parser"
)

func TestEmitNumberWhereAndExpression(t *testing.T) {
	m, err := parser.LoadSemanticModel("../../examples/orders.semantic.yaml")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Emit(m)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		"type: number",
		"Lost Customer",
		"{revenue} - {lost_revenue}",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("cube yaml missing %q\n%s", want, s)
		}
	}
}

func TestEmitExpressionWhereAndLTM(t *testing.T) {
	m := &parser.SemanticModel{
		Metadata: parser.Metadata{Name: "movements"},
		Spec: parser.SemanticModelSpec{
			Measures: []parser.Measure{
				{Name: "closing", Aggregation: "sum", Field: "mrr"},
				{Name: "opening", Aggregation: "sum", Field: "prev_mrr"},
				{
					Name:       "lost_diff",
					Expression: "closing - opening",
					Where:      []parser.Predicate{{Dimension: "movement", Operator: "eq", Value: "Lost"}},
				},
				{Name: "revenue_ltm", Aggregation: "ltm", Field: "amount"},
			},
		},
	}
	b, err := Emit(m)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		"{closing} - {opening}",
		"movement",
		"Lost",
		"rolling_window:",
		`trailing: 12 month`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("cube yaml missing %q\n%s", want, s)
		}
	}
}
