package collection

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Expression is a compiled Expr (github.com/expr-lang/expr) boolean condition
// over one document. MEANS.md app.views filters and the Where options of List
// and Search share this environment:
//
//   - id, title, body: strings (body is the exact Markdown body)
//   - created_at, updated_at: RFC 3339 strings, or nil for legacy documents
//   - data: frontmatter.data; integers are int64, other numbers float64
//   - now(): the evaluation time, fixed for one operation
//
// Expr builtins such as date, duration, in, not, and ?? are available.
// An Expression is safe for concurrent use.
type Expression struct {
	program *vm.Program
}

// CompileExpression compiles source. A blank source matches every document.
// Expressions that are statically non-boolean fail to compile.
func CompileExpression(source string) (*Expression, error) {
	if strings.TrimSpace(source) == "" {
		source = "true"
	}
	program, err := expr.Compile(source, expr.Env(map[string]any{
		"id": "", "title": "", "body": "", "data": map[string]any{}, "now": func() time.Time { return time.Time{} },
		CreatedAtField: nil, UpdatedAtField: nil,
	}), expr.AsBool(), expr.MaxNodes(1000), expr.Optimize(false))
	if err != nil {
		return nil, err
	}
	return &Expression{program: program}, nil
}

// Match evaluates the expression for a document's frontmatter (id, title, timestamps, data)
// and body. Runtime errors and non-boolean results return an error.
func (e *Expression) Match(frontmatter map[string]any, body string, now time.Time) (bool, error) {
	data, err := exprValue(frontmatter["data"])
	if err != nil {
		return false, err
	}
	if data == nil {
		data = map[string]any{}
	}
	machine := vm.VM{MemoryBudget: 100000}
	result, err := machine.Run(e.program, map[string]any{
		"id": frontmatter["id"], "title": frontmatter["title"], "data": data, "body": body,
		CreatedAtField: frontmatter[CreatedAtField], UpdatedAtField: frontmatter[UpdatedAtField],
		"now": func() time.Time { return now },
	})
	if err != nil {
		return false, err
	}
	match, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("expression returned %T, expected boolean", result)
	}
	return match, nil
}

// exprValue adapts a copy for Expr; exact filters retain JSON semantics.
func exprValue(value any) (any, error) {
	switch value := value.(type) {
	case json.Number:
		if n, err := value.Int64(); err == nil {
			return n, nil
		}
		if !strings.ContainsAny(string(value), ".eE") {
			return nil, fmt.Errorf("integer %s exceeds Expr's int64 range", value)
		}
		return value.Float64()
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			converted, err := exprValue(v)
			if err != nil {
				return nil, err
			}
			out[k] = converted
		}
		return out, nil
	case []any:
		out := make([]any, len(value))
		for i, v := range value {
			converted, err := exprValue(v)
			if err != nil {
				return nil, err
			}
			out[i] = converted
		}
		return out, nil
	default:
		return value, nil
	}
}
