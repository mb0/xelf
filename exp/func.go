package exp

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lex"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// Func is the common type for all function literals and implements both literal and resolver.
// It consists of a signature and body. A func is consider to a zero value if the body is nil,
// any other body value must be a valid function body. If the body implements bfr writer
// it is called for printing the body expressions.
// Resolution handles reference and delegates expression resolution to the body.

var funcSig = MustSig("<form _ args?; @>")

// FuncLayout matches arguments of x to the parameters of f and returns a layout or an error.
func FuncLayout(sig typ.Type, els []El) (*Layout, error) {
	lo, err := FormLayout(funcSig, els)
	if err != nil {
		return nil, err
	}
	tags := lo.Tags(0)
	params := sig.Params
	if len(params) > 0 {
		params = params[:len(params)-1]
	}
	if len(params) == 0 {
		if len(tags) > 0 {
			return nil, cor.Errorf("unexpected arguments %s", sig)
		}
		return &Layout{}, nil
	}
	vari := isVariadic(params)
	var tagged bool
	args := make([][]El, len(params))
	for i, tag := range tags {
		idx := i
		if tag.Name == "" {
			if tagged {
				return nil,
					cor.Errorf("positional param after tag parameter in %s", sig)
			}
			if idx >= len(args) {
				if vari {
					idx = len(args) - 1
					args[idx] = append(args[idx], tag.El)
					continue
				}
				return nil, cor.Errorf("unexpected arguments %s", sig)
			}
		} else if tag.Name == "::" {
			if vari {
				idx = len(args) - 1
				args[idx] = append(args[idx], tag.El)
				continue
			}
			return nil, cor.Errorf("unexpected arguments %s", sig)
		} else {
			tagged = true
			_, idx, err = sig.ParamByKey(tag.Key())
			if err != nil {
				return nil, err
			}
		}
		if len(args[idx]) > 0 {
			return nil, cor.Errorf("duplicate parameter %s", params[idx].Name)
		}
		args[idx] = []El{tag.El}
	}
	for i, pa := range params {
		arg := args[i]
		if len(arg) == 0 {
			if pa.Opt() {
				continue
			}
			return nil, cor.Errorf("missing non optional parameter %s", pa.Name)
		}
	}
	return &Layout{sig, args}, nil
}
func ReslFuncArgs(p *Prog, env Env, c *Call) (*Layout, error) {
	params := c.Spec.Arg()
	vari := isVariadic(params)
	for i, param := range params {
		a := c.Groups[i]
		if len(a) == 0 { // skip; nothing to resolve
			continue
		}
		if i == len(params)-1 && vari && len(a) > 1 {
			ll, err := reslListArr(p, env, a, param.Type)
			if err != nil {
				return nil, err
			}
			a[0] = ll
			a = a[:1]
			break
		}
		if len(a) > 1 {
			return nil, cor.Errorf(
				"multiple arguments for non variadic parameter %s", param.Name)
		}
		el, err := p.Resl(env, a[0], param.Type)
		if err != nil {
			return nil, err
		}
		a[0] = el
	}
	return &c.Layout, nil
}

func EvalFuncArgs(p *Prog, env Env, c *Call) (*Layout, error) {
	params := c.Spec.Arg()
	vari := isVariadic(params)
	for i, param := range params {
		a := c.Groups[i]
		if len(a) == 0 { // skip; nothing to resolve
			continue
		}
		if i == len(params)-1 && vari && len(a) > 1 {
			ll, err := evalListArr(p, env, param.Type.Elem(), a)
			if err != nil {
				return nil, err
			}
			c.Groups[i] = []El{ll}
			break
		}
		el, err := p.Eval(env, a[0], param.Type)
		if err != nil {
			return nil, err
		}
		if at, ok := el.(*Atom); ok {
			if param.Type != typ.Void && param.Type != typ.Any {
				at.Lit, err = lit.Convert(at.Lit, param.Type, 0)
				if err != nil {
					return nil, err
				}
			}
			el = at
		}
		a[0] = el
	}
	return &c.Layout, nil
}

func reslListArr(p *Prog, env Env, args []El, t typ.Type) (El, error) {
	con := Lookup(env, "con")
	args = append([]El{&Atom{Lit: t}}, args...)
	c, err := p.NewCall(con.Lit.(*Spec), args, lex.Src{})
	if err != nil {
		return nil, err
	}
	return p.Resl(env, c, t)
}

func evalListArr(p *Prog, env Env, et typ.Type, args []El) (*Atom, error) {
	els, err := p.EvalAll(env, args, et)
	if err != nil {
		return nil, err
	}
	res := make([]lit.Lit, 0, len(els))
	for _, el := range els {
		l := el.(*Atom).Lit
		if et != typ.Any {
			l, err = lit.Convert(l, et, 0)
			if err != nil {
				return nil, err
			}
		}
		res = append(res, l)
	}
	return &Atom{Lit: &lit.List{et, res}}, nil
}

func isVariadic(ps []typ.Param) bool {
	if len(ps) != 0 {
		switch ps[len(ps)-1].Type.Kind & typ.SlotMask {
		case typ.KindIdxr, typ.KindList:
			return true
		}
	}
	return false
}
