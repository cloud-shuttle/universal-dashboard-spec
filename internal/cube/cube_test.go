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
