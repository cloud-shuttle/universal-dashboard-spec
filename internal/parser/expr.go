package parser

import (
	"fmt"
	"unicode"
)

// Expr is a restricted arithmetic expression over measure names.
// A leaf sets Name or Num. A node sets Op to "+" "-" "*" "/" or "neg".
type Expr struct {
	Op    string
	Name  string
	Num   string
	Left  *Expr
	Right *Expr
}

// ParseExpression parses identifiers, numbers, + - * /, parentheses, and unary minus.
// Function calls and any other syntax are rejected.
func ParseExpression(s string) (*Expr, error) {
	p := &exprParser{s: s}
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.i < len(p.s) {
		return nil, fmt.Errorf("expression %q: unexpected %q", s, p.s[p.i:])
	}
	return e, nil
}

type exprParser struct {
	s string
	i int
}

func (p *exprParser) skip() {
	for p.i < len(p.s) && unicode.IsSpace(rune(p.s[p.i])) {
		p.i++
	}
}

func (p *exprParser) parseExpr() (*Expr, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for {
		p.skip()
		if p.i >= len(p.s) || (p.s[p.i] != '+' && p.s[p.i] != '-') {
			return left, nil
		}
		op := string(p.s[p.i])
		p.i++
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &Expr{Op: op, Left: left, Right: right}
	}
}

func (p *exprParser) parseTerm() (*Expr, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for {
		p.skip()
		if p.i >= len(p.s) || (p.s[p.i] != '*' && p.s[p.i] != '/') {
			return left, nil
		}
		op := string(p.s[p.i])
		p.i++
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &Expr{Op: op, Left: left, Right: right}
	}
}

func (p *exprParser) parseFactor() (*Expr, error) {
	p.skip()
	if p.i >= len(p.s) {
		return nil, fmt.Errorf("expression %q: unexpected end", p.s)
	}
	if p.s[p.i] == '-' {
		p.i++
		inner, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &Expr{Op: "neg", Left: inner}, nil
	}
	if p.s[p.i] == '(' {
		p.i++
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		p.skip()
		if p.i >= len(p.s) || p.s[p.i] != ')' {
			return nil, fmt.Errorf("expression %q: missing )", p.s)
		}
		p.i++
		return inner, nil
	}
	if p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		start := p.i
		for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
		}
		if p.i < len(p.s) && p.s[p.i] == '.' {
			p.i++
			if p.i >= len(p.s) || p.s[p.i] < '0' || p.s[p.i] > '9' {
				return nil, fmt.Errorf("expression %q: bad number", p.s)
			}
			for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
				p.i++
			}
		}
		return &Expr{Num: p.s[start:p.i]}, nil
	}
	if isIdentStart(p.s[p.i]) {
		start := p.i
		p.i++
		for p.i < len(p.s) && isIdentCont(p.s[p.i]) {
			p.i++
		}
		name := p.s[start:p.i]
		// A following "(" is a function call, which this vocabulary does not allow.
		p.skip()
		if p.i < len(p.s) && p.s[p.i] == '(' {
			return nil, fmt.Errorf("expression %q: function calls are not allowed", p.s)
		}
		return &Expr{Name: name}, nil
	}
	return nil, fmt.Errorf("expression %q: unexpected %q", p.s, p.s[p.i:])
}

func isIdentStart(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isIdentCont(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

// FormatExpr renders e with a caller-supplied identifier spelling.
func FormatExpr(e *Expr, ident func(name string) string) string {
	switch {
	case e == nil:
		return ""
	case e.Name != "":
		return ident(e.Name)
	case e.Num != "":
		return e.Num
	case e.Op == "neg":
		return "-(" + FormatExpr(e.Left, ident) + ")"
	default:
		return "(" + FormatExpr(e.Left, ident) + " " + e.Op + " " + FormatExpr(e.Right, ident) + ")"
	}
}
