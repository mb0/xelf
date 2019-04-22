package exp

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

func init() {
	var rest2 = []typ.Param{
		{Name: "a", Type: typ.Any},
		{Name: "b", Type: typ.Any},
		{Name: "plain", Type: typ.List(typ.Any)},
		{Type: typ.Bool},
	}
	core.add("eq", rest2, rslvEq)
	core.add("ne", rest2, rslvNe)
	core.add("equal", rest2, rslvEqual)

	var ab = []typ.Param{
		{Name: "a", Type: typ.Any},
		{Name: "b", Type: typ.Idxer},
		{Type: typ.Bool},
	}
	core.add("in", ab, rslvIn)
	core.add("ni", ab, rslvNi)

	core.add("lt", rest2, rslvLt)
	core.add("ge", rest2, rslvGe)
	core.add("gt", rest2, rslvGt)
	core.add("le", rest2, rslvLe)
}

// rslvEq returns a bool whether the arguments are equivalent literals.
// The result is negated, if the expression symbol is 'ne'.
// (form +a +b any +rest? list - bool)
func rslvEq(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return resolveBinaryComp(c, env, e, true, lit.Equiv)
}

func rslvNe(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	res, err := resolveBinaryComp(c, env, e, true, lit.Equiv)
	if err != nil {
		return res, err
	}
	return !res.(lit.Bool), nil
}

// rslvEqual returns a bool whether the arguments are same types or same literals.
// (form +a +b any +rest? list - bool)
func rslvEqual(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return resolveBinaryComp(c, env, e, true, lit.Equal)
}
func rslvIn(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return inOrNi(c, env, e, false)
}
func rslvNi(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return inOrNi(c, env, e, true)
}

func inOrNi(c *Ctx, env Env, e *Call, neg bool) (El, error) {
	lo, err := ResolveArgs(c, env, e)
	if err != nil {
		return e, err
	}
	a := lo.Arg(0).(Lit)
	list, ok := lo.Arg(1).(lit.Indexer)
	if !ok {
		return nil, cor.Errorf("expect idxer got %s", lo.Arg(1).Typ())
	}
	var found bool
	err = list.IterIdx(func(idx int, el Lit) error {
		if found = lit.Equal(el, a); found {
			return lit.BreakIter
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if neg {
		return lit.Bool(!found), nil
	}
	return lit.Bool(found), nil
}

// rslvLt returns a bool whether the arguments are monotonic increasing literals.
// Or the inverse, if the expression symbol is 'ge'.
// (form +a +b any +rest? list - bool)
func rslvLt(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return resolveBinaryComp(c, env, e, false, func(a, b Lit) bool {
		res, ok := lit.Less(a, b)
		return ok && res
	})
}

func rslvGe(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return resolveBinaryComp(c, env, e, false, func(a, b Lit) bool {
		res, ok := lit.Less(a, b)
		return ok && !res
	})
}

// rslvGt returns a bool whether the arguments are monotonic decreasing literals.
// Or the inverse, if the expression symbol is 'le'.
// (form +a +b any +rest? list - bool)
func rslvGt(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return resolveBinaryComp(c, env, e, false, func(a, b Lit) bool {
		res, ok := lit.Less(b, a)
		return ok && res
	})
}
func rslvLe(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return resolveBinaryComp(c, env, e, false, func(a, b Lit) bool {
		res, ok := lit.Less(b, a)
		return ok && !res
	})
}

type cmpf = func(a, b Lit) bool

func resolveBinaryComp(c *Ctx, env Env, e *Call, sym bool, cmp cmpf) (El, error) {
	_, err := LayoutArgs(e.Spec.Arg(), e.Args)
	if err != nil {
		return nil, err
	}
	var res, init bool
	var unres []El
	var last Lit
	for _, arg := range e.Args {
		arg, err = c.Resolve(env, arg, typ.Void)
		if err == ErrUnres {
			if !c.Part {
				return e, err
			}
			if len(unres) == 0 {
				unres = make([]El, 0, len(e.Args))
				if res {
					init = true
					unres = append(unres, last)
				}
			}
			res = false
			unres = append(unres, arg)
			continue
		}
		if err != nil {
			return nil, err
		}
		el := arg.(Lit)
		if last != nil {
			if !cmp(last, el) {
				return lit.False, nil
			}
		}
		if !res && ((!sym || !init) && len(unres) > 0) || len(unres) == 1 {
			unres = append(unres, el)
		}
		last = el
		res = true
	}
	if len(unres) != 0 {
		e.Args = unres
		return e, ErrUnres
	}
	return lit.True, nil
}
