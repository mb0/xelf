package exp

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

// Layout is a helper to validate form expression arguments.
//
// We distinguish between tag and declaration expressions, and all other elements, that we call the
// plain elements in the context of layouts.
//
// The layout is formalizes by the parameter signature. It uses a number of special parameter names,
// that indicate to the layout how arguments must be parsed. A full form consisting of all possible
// kinds of accepted elements, is: (form 'full' +args +decls +tail)
//
// These are the recognised parameter names:
//    plain or rest accepts any number of plain elements.
//    tags accepts any number of tag expressions.
//    args or tail  accepts any number of leading plain elements and then tag expression.
//    unis accepts declarations with at most one argument.
//    decls accepts declarations with multiple arguments.
//
// With some caveats:
//    Each special parameter name can occur only once.
//    Plain and rest or args and tail cannot follow each other.
//    Plain or rest can not follow args or tail.
//    Only one of either unis or decls can occur.
//
// Explicit parameters are plain elements and can only occur in front or instead plain arguments.
//
// The parameter types give hints at what types are accepted. The special parameters can only use
// container types. The unis and decls parameters expect a keyer type, while all others accpect an
// idxer type. If the type is omitted, the layout will not resolve or check that parameter.
//
// TODO:
//    add type validation when hints are provided
type Layout struct {
	ps   []typ.Param
	args [][]El
}

func (l *Layout) Args(idx int) []El {
	if l == nil || idx < 0 || idx >= len(l.args) {
		return nil
	}
	return l.args[idx]
}

func (l *Layout) Arg(idx int) El {
	args := l.Args(idx)
	if len(args) == 0 {
		return nil
	}
	return args[0]
}

func (l *Layout) Tags(idx int) ([]*Named, error) {
	args := l.Args(idx)
	res := make([]*Named, 0, len(args))
	for _, arg := range args {
		switch arg.Typ() {
		case typ.Tag:
			res = append(res, arg.(*Named))
		default:
			res = append(res, &Named{El: arg})
		}
	}
	return res, nil
}

func (l *Layout) Decls(idx int) ([]*Named, error) {
	args := l.Args(idx)
	res := make([]*Named, 0, len(args))
	for _, arg := range args {
		switch arg.Typ() {
		case typ.Decl:
			res = append(res, arg.(*Named))
		default:
			return nil, cor.Errorf("unexpected decl element %s", arg)
		}
	}
	return res, nil
}

func (l *Layout) Unis(idx int) ([]*Named, error) {
	args := l.Args(idx)
	res := make([]*Named, 0, len(args))
	var naked int
	for _, arg := range args {
		switch arg.Typ() {
		case typ.Decl:
			n := arg.(*Named)
			res = append(res, n)
			if n.El == nil {
				naked++
				continue
			}
			for naked > 0 {
				res[len(res)-naked-1].El = n.El
				naked--
			}
		default:
			if naked == 0 {
				return nil, cor.Errorf("unexpected uni element %T %s", arg, arg)
			}
			for naked > 0 {
				res[len(res)-naked].El = arg
				naked--
			}
		}
	}
	return res, nil
}

func (l *Layout) Resolve(c *Ctx, env Env) error {
	if l == nil {
		return nil
	}
	var res error
	for i, p := range l.ps {
		if p.Type == typ.Void {
			continue
		}
		args := l.args[i]
		if len(args) == 0 {
			continue
		}
		switch p.Name {
		case "plain", "rest", "tags", "tail", "args", "decls", "unis":
			args, err := c.ResolveAll(env, args, typ.Void)
			if err != nil {
				if err == ErrUnres {
					res = err
					continue
				}
				return err
			}
			if !c.Part {
				l.args[i] = args
			}
		default: // explicit param
			el, err := c.Resolve(env, args[0], p.Type)
			if err != nil {
				if err == ErrUnres {
					res = err
					continue
				}
				return err
			}
			cmp := typ.Compare(el.Typ(), p.Type)
			if cmp < typ.LvlCheck {
				return cor.Errorf("cannot convert %s to %s", el.Typ(), p.Type)
			}
			if c.Part {
				args[0] = el
			} else {
				l.args[i] = []El{el}
			}
		}
	}
	return res
}

func LayoutArgs(params []typ.Param, args []El) (*Layout, error) {
	if len(params) == 0 {
		if len(args) > 0 {
			return nil, cor.Error("unexpected argument count")
		}
		return &Layout{}, nil
	}
	res := make([][]El, 0, len(params))
	var tmp []El
Loop:
	for _, p := range params {
		// check kind of parameter and consume matching args
		tmp = nil
		switch p.Name {
		case "plain":
			tmp, args = consumePlain(args, tmp)
		case "rest":
			for len(args) > 0 {
				tmp, args = consumePlain(args, tmp)
				tmp, args = consumeTags(args, tmp)
				tmp, args = consumeDecls(args, tmp)
			}
		case "tail", "args":
			tmp, args = consumePlain(args, tmp)
			tmp, args = consumeTags(args, tmp)
		case "tags":
			tmp, args = consumeTags(args, tmp)
		case "decls":
			tmp, args = consumeDecls(args, tmp)
		case "unis":
			tmp, args = consumeUnis(args)
		default: // explicit param
			if len(args) == 0 {
				if !p.Opt() {
					return nil, cor.Errorf("missing argument for %s", p)
				}
				continue
			}
			if _, _, _, ok := isSpecial(args[0], ":+-;"); ok {
				if !p.Opt() {
					break Loop
				}
			} else {
				tmp, args = args[:1], args[1:]
			}
		}
		res = append(res, tmp)
	}
	// at this point all arguments should have been consumed
	if len(args) > 0 {
		return nil, cor.Errorf("unexpected tail element %s", args[0])
	}
	return &Layout{params, res}, nil
}

func ResolveArgs(c *Ctx, env Env, e *Call) (*Layout, error) {
	lo, err := LayoutArgs(e.Spec.Arg(), e.Args)
	if err != nil {
		return lo, err
	}
	err = lo.Resolve(c, env)
	return lo, err
}
func consumeArg(es []El) (El, []El) {
	if len(es) != 0 {
		e := es[0]
		if _, _, _, ok := isSpecial(e, ":+-;"); !ok {
			return e, es[1:]
		}
	}
	return nil, es
}
func consumeTag(es []El) (El, []El) {
	if len(es) != 0 {
		e := es[0]
		if t, s, a, ok := isSpecial(e, ":+-;"); ok {
			if s[0] != ':' {
				return nil, es
			}
			tag := &Named{Name: s}
			es = es[1:]
			if t == typ.Dyn || len(a) > 0 {
				els, a := consumePlain(a, nil)
				els, a = consumeTags(a, els)
				els, a = consumeDecls(a, els)
				tag.El = &Dyn{Els: els}
			} else {
				tag.El, es = consumeArg(es)
			}
			return tag, es
		}
	}
	return nil, es
}

func consumeDecl(es []El, uni bool) (El, []El) {
	if len(es) == 0 {
		return nil, nil
	}
	e := es[0]
	if t, s, a, ok := isSpecial(e, "+-;"); ok {
		if s[0] == ';' {
			return nil, es[1:]
		}
		d := &Named{Name: s}
		es = es[1:]
		var els []El
		if t == typ.Dyn || len(a) > 0 {
			els, a = consumePlain(a, els)
			els, a = consumeTags(a, els)
			els, a = consumeDecls(a, els)
			if t == typ.Dyn {
				d.El = &Dyn{Els: els}
				return d, es
			}
		} else if uni {
			d.El, es = consumeArg(es)
		} else {
			els, es = consumePlain(es, els)
			els, es = consumeTags(es, els)
		}
		switch len(els) {
		case 0:
		case 1:
			d.El = els[0]
		default:
			d.El = &Dyn{Els: els}
		}
		return d, es
	}
	return nil, es
}

func consumePlain(es []El, res []El) ([]El, []El) {
	var e El
	for len(es) > 0 {
		e, es = consumeArg(es)
		if e != nil {
			res = append(res, e)
			continue
		}
		break
	}
	return res, es
}
func consumeTags(es []El, res []El) ([]El, []El) {
	var e El
	for len(es) > 0 {
		e, es = consumeTag(es)
		if e != nil {
			res = append(res, e)
			continue
		}
		break
	}
	return res, es
}
func consumeDecls(es []El, res []El) ([]El, []El) {
	var e El
	for len(es) > 0 {
		e, es = consumeDecl(es, false)
		if e != nil {
			res = append(res, e)
			continue
		}
		break
	}
	return res, es
}
func consumeUnis(es []El) (res, _ []El) {
	var e El
	for len(es) > 0 {
		e, es = consumeDecl(es, true)
		if e != nil {
			res = append(res, e)
			continue
		}
		break
	}
	return res, es
}

func isSpecial(e El, pre string) (t Type, s string, a []El, ok bool) {
	switch t = e.Typ(); t {
	case typ.Dyn:
		switch d := e.(type) {
		case *Dyn:
			if len(d.Els) != 0 {
				s, ok = isSpecialSym(d.Els[0], pre)
				a = d.Els[1:]
			}
		}
	case typ.Tag, typ.Decl:
		v := e.(*Named)
		d := v.Dyn()
		s, ok = v.Name, true
		if d != nil {
			t, a = typ.Dyn, d.Els
		} else {
			a = v.Args()
		}
	}
	return
}
func isSpecialSym(e El, pre string) (string, bool) {
	switch v := e.(type) {
	case *Sym:
		return isSpecialName(v.Name, pre)
	}
	return "", false
}
func isSpecialName(name string, pre string) (string, bool) {
	return name, name != "" && strings.IndexByte(pre, name[0]) != -1
}
