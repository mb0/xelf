package std

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

var errConType = cor.StrError("the 'con' expression must start with a type")

// dynSpec resolves a dynamic expressions. If the first element resolves to a type it is
// resolves as the 'con' expression. If it is a literal it selects an appropriate combine
// expression for that literal. The time and uuid literals have no such combine expression.
var dynSpec = core.impl("(form 'dyn' @ :rest? list @)",
	func(c *Ctx, env Env, e *Call, lo *Layout, hint Type) (_ El, err error) {
		if len(e.Args) == 0 {
			return typ.Void, nil
		}
		return defaultDyn(c, env, &exp.Dyn{Els: e.Args}, hint)
	})

func defaultDyn(c *Ctx, env Env, d *exp.Dyn, hint Type) (_ El, err error) {
	if len(d.Els) == 0 {
		return typ.Void, nil
	}
	fst := d.Els[0]
	switch t := fst.Typ(); t.Kind {
	case typ.KindSym, typ.KindCall:
		fst, err = c.Resolve(env, fst, typ.Void)
	case typ.KindDyn:
		v, _ := fst.(*exp.Dyn)
		if len(v.Els) == 0 {
			return typ.Void, nil
		}
		fst, err = defaultDyn(c, env, v, typ.Void)
	}
	if err != nil {
		return d, err
	}
	var spec *exp.Spec
	var sym string
	args := d.Els
	switch t := fst.Typ(); t.Kind & typ.MaskRef {
	case typ.KindTyp:
		if a, ok := fst.(*exp.Atom); ok {
			fst = a.Lit
		}
		tt := fst.(Type)
		if tt == typ.Void {
			return fst, nil
		}
		if tt == typ.Bool {
			spec, args = boolSpec, args[1:]
		} else {
			sym = "con"
		}
	case typ.KindFunc, typ.KindForm:
		r, ok := fst.(*exp.Spec)
		if ok {
			spec, args = r, args[1:]
		}
	default:
		if len(d.Els) == 1 && t.Kind&typ.KindAny != 0 {
			if a, ok := fst.(*exp.Atom); ok {
				fst = a.Lit
			}
			return fst, nil
		}
		switch t.Kind & typ.MaskElem {
		case typ.KindBool:
			sym = "and"
		case typ.KindNum, typ.KindInt, typ.KindReal, typ.KindSpan:
			sym = "add"
		case typ.KindChar, typ.KindStr, typ.KindRaw:
			sym = "cat"
		case typ.KindIdxr, typ.KindList:
			sym = "apd" // TODO maybe cat
		case typ.KindKeyr, typ.KindDict, typ.KindRec:
			sym = "set" // TODO maybe merge
		}
	}
	if sym != "" {
		def := exp.LookupSupports(env, sym, '~')
		if def != nil {
			spec, _ = def.Lit.(*exp.Spec)
		}
	}
	if spec != nil {
		t := c.Inst(spec.Type)
		return spec.Resolve(c, env, &Call{Spec: spec, Type: t, Args: args}, hint)
	}
	return nil, cor.Errorf("unexpected first argument %[1]T %[1]s in dynamic expression\n%s %s",
		fst, sym, fst.Typ())
}

// conSpec is a type conversion or constructor and must start with a type. It has four forms:
//    Without further arguments it returns the zero literal for that type.
//    With one literal compatible to that type it returns the converted literal.
//    For keyer types one or more declarations are set.
//    For idxer types one ore more literals are appended.
var conSpec = core.impl("(form 'con' typ :args? list :unis? dict @)",
	func(c *Ctx, env Env, e *Call, lo *Layout, hint Type) (El, error) {
		// resolve all arguments
		err := lo.Resolve(c, env, hint)
		if err != nil {
			t, ok := lo.Arg(0).(Type)
			if ok && hint != typ.Void {
				_, err := typ.Unify(&c.Ctx, hint, t)
				if err == nil {
					e.Type = c.Apply(e.Type)
				}
			}
			return e, err
		}
		t, ok := lo.Arg(0).(Type)
		if !ok {
			return nil, errConType
		}
		if hint != typ.Void {
			typ.Unify(&c.Ctx, hint, t)
			e.Type = c.Apply(e.Type)
		}
		if t == typ.Void { // just in case we have a dynamic comment
			return typ.Void, nil
		}
		if hint != typ.Void {
			_, err = typ.Unify(&c.Ctx, hint, t)
			if err != nil {
				return typ.Void, err
			}
		}
		args := lo.Args(1)
		decls, err := lo.Unis(2)
		if err != nil {
			return nil, err
		}
		// first rule: return zero literal
		if len(args) == 0 && len(decls) == 0 {
			return lit.Zero(t), nil
		}
		// second rule: convert compatible literals
		if len(args) == 1 && len(decls) == 0 {
			fst := args[0].(Lit)
			res, err := lit.Convert(fst, t, 0)
			if err == nil {
				return res, nil
			}
		}
		// third rule: set declarations
		if t.Kind&typ.KindKeyr != 0 {
			res := deopt(lit.Zero(t)).(lit.Keyer)
			for _, d := range decls {
				el, ok := d.Arg().(Lit)
				if !ok {
					return nil, cor.Errorf("want literal in declaration got %s", d.El)
				}
				_, err = res.SetKey(d.Key(), el)
				if err != nil {
					return nil, err
				}
			}
			return res, nil
		}
		// fourth rule: append list
		if ok && t.Kind&typ.KindIdxr != 0 {
			res := deopt(lit.Zero(t)).(lit.Indexer)
			apd, _ := res.(lit.Appender)
			for i, a := range args {
				el, ok := a.(Lit)
				if !ok {
					return nil, cor.Error("want literal in argument list")
				}
				if apd != nil { // list uses append
					apd, err = apd.Append(el)
				} else { // otherwise its a record literal set by index
					_, err = res.SetIdx(i, el)
				}
				if err != nil {
					return nil, err
				}
			}
			if apd != nil {
				return apd, nil
			}
			return res, nil
		}
		return nil, cor.Error("not implemented")
	})