package workflow

import (
	"context"
	"fmt"
	"github.com/PaesslerAG/gval"
	"strings"
)

type (
	expression struct {
		// where to assign the evaluated expression
		name string

		// expression
		expr string

		// expression, ready to be executed
		eval gval.Evaluable
	}

	expressions struct {
		lang gval.Language
		set  []*expression
	}
)

func Expression(lang gval.Language, dst, expr string) (e *expression, err error) {
	e = &expression{name: dst, expr: expr}

	if e.eval, err = lang.NewEvaluable(expr); err != nil {
		return nil, fmt.Errorf("can not parse expression %s: %w", expr, err)
	}

	return e, nil
}

func Expressions(lang gval.Language, ee ...*expression) *expressions {
	return &expressions{
		lang: lang,
		set:  ee,
	}
}

func (ee *expressions) Set(dst, expr string) error {
	var (
		e, err = Expression(ee.lang, dst, expr)
	)

	if err != nil {
		return err
	}

	for i := range ee.set {
		if ee.set[i].name == dst {
			ee.set[i] = e
			return nil
		}
	}

	ee.set = append(ee.set, e)
	return nil
}

func (ee *expressions) Exec(ctx context.Context, r *ExecRequest) (ExecResponse, error) {
	if result, err := ee.Eval(ctx, r.Scope); err != nil {
		return nil, err
	} else {
		return r.Scope.Merge(result), nil
	}
}

func (ee *expressions) Eval(ctx context.Context, in Variables) (Variables, error) {
	var (
		err error
		// Copy/create scope
		scope = Variables.Merge(in)
		out   = Variables{}
	)

	for _, e := range ee.set {
		if strings.Contains(e.name, ".") {
			// handle property setting
			return nil, fmt.Errorf("dot/prop setting not supported at the moment")
		}

		if scope[e.name], err = e.eval(ctx, scope); err != nil {
			return nil, fmt.Errorf("could not evaluate %q for %q: %w", e.expr, e.name, err)
		}

		out[e.name] = scope[e.name]
	}

	return out, nil
}
