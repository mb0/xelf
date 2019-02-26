package exp

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// Resolve creates a new non-executing resolution context and resolves x with with given env.
func Resolve(env Env, x El) (El, error) { return (&Ctx{Part: true}).Resolve(env, x) }

// Execute creates a new executing resolution context and evaluates x with with given env.
func Execute(env Env, x El) (El, error) { return (&Ctx{Exec: true}).Resolve(env, x) }

// ResolveAll tries to resolve each element in xs in place and returns the first error if any.
func (c *Ctx) ResolveAll(env Env, els []El) ([]El, error) {
	var res error
	xs := els
	if !c.Part {
		xs = make([]El, len(els))
	}
	for i, x := range els {
		r, err := c.Resolve(env, x)
		xs[i] = r
		if err != nil {
			if !c.Exec && err == ErrUnres {
				res = err
				continue
			}
			return nil, err
		}
	}
	return xs, res
}

// Resolve resolves x within env and returns the result or an error.
//
// This method will not resolve any element itself but instead tries to look up an applicable
// resolver in the environment. If it cannot find a resolver it will add the element to the
// context's unresolved slice.
// The resolver implementations usually use this method either directly or indirectly to resolve
// arguments, which are then again added to the unresolved elements when appropriate.
func (c *Ctx) Resolve(env Env, x El) (res El, err error) {
	var xx *Expr
	switch v := x.(type) {
	case nil:
		return typ.Void, nil
	case Type: // resolve type references
		last := v.Last()
		if last.Kind&typ.FlagRef != 0 {
			v, err = c.resolveTypRef(env, v, last)
			if err == ErrUnres {
				c.Unres = append(c.Unres, x)
				return x, err
			}
		} else if last.Kind == typ.KindFunc {
			// TODO resolve func signatures
		}
		return v, err
	case Lit: // already resolved
		return v, nil
	case *Ref:
		return c.resolveRef(env, v)
	case Tag:
		_, err = c.ResolveAll(env, v.Args)
		return v, err
	case Decl:
		_, err = c.ResolveAll(env, v.Args)
		return v, err
	case *Expr:
		xx = v
	case Dyn:
		xx = &Expr{Ref{Name: "dyn"}, v, Lookup(env, "dyn")}
	default:
		return x, cor.Errorf("unexpected expression %T %v", x, x)
	}
	if xx == nil {
		c.Unres = append(c.Unres, x)
		return x, ErrUnres
	}
	// resolvers add to unres list themselves
	return xx.Resolve(c, env, xx)
}

func (c *Ctx) resolveRef(env Env, ref *Ref) (El, error) {
	sym := ref.Key()
	r, name, path, err := findResolver(env, sym)
	if err != nil {
		return ref, err
	}
	if r == nil {
		return ref, ErrUnres
	}
	if sym == name {
		return r.Resolve(c, env, ref)
	}
	tmp := &Ref{Name: name}
	res, err := r.Resolve(c, env, tmp)
	if err != nil {
		return ref, err
	}
	if path == "" {
		return res, nil
	}
	return lit.Select(res.(Lit), path)
}

func (c *Ctx) resolveTypRef(env Env, t Type, last Type) (_ Type, err error) {
	k := last.Kind
	if t.Info == nil || t.Info.Ref == "" {
		if k != typ.FlagRef {
			return t, cor.Errorf("unnamed %s not allowed", k)
		}
		// TODO infer type
		return t, ErrUnres
	}
	key := t.Info.Key()
	switch k {
	case typ.KindFlag, typ.KindEnum, typ.KindRec:
		// return already resolved schema types, otherwise add schema prefix '~'
		if len(t.Fields) > 0 || len(t.Consts) > 0 {
			return t, nil
		}
		key = "~" + key
	}
	res, err := c.resolveRef(env, &Ref{Name: key})
	if err != nil {
		return t, err
	}
	et, err := elType(res)
	if err != nil {
		return t, err
	}
	return replaceRef(t, et)
}

func findResolver(env Env, sym string) (r Resolver, name, path string, err error) {
	if sym == "" {
		return nil, "", "", cor.Error("empty symbol")
	}
	// check for relative paths prefixes
	var lookup bool
	switch x := sym[0]; x {
	case '~', '$', '/':
		return LookupSupports(env, sym, x), sym, "", nil
	case '.':
		sym = sym[1:]
		for len(sym) > 0 && sym[0] == '.' {
			sym = sym[1:]
			env = env.Parent()
			if env == nil {
				return nil, "", "", cor.Errorf("no env found for prefix %q", x)
			}
		}
		if len(sym) > 0 && sym[0] == '?' {
			lookup = true
			sym = sym[1:]
		}
	default:
		lookup = true
	}
	// check for path
	idx := strings.IndexByte(sym, '.')
	if idx > 0 {
		sym, path = sym[:idx], sym[idx+1:]
	}
	if lookup {
		r = Lookup(env, sym)
	} else {
		r = env.Get(sym)
	}
	return r, sym, path, nil
}

func elType(el El) (Type, error) {
	switch et := el.(type) {
	case Type:
		return et, nil
	case Lit:
		return et.Typ(), nil
	case *Ref:
		if et.Type != typ.Void {
			return et.Type, nil
		}
	case *Expr:
		if et.Type != typ.Void {
			return et.Type, nil
		}
	}
	return typ.Void, ErrUnres
}

func replaceRef(t, el Type) (Type, error) {
	var mask, shift typ.Kind
	for shift = 0; ; shift += typ.SlotSize {
		k := t.Kind >> shift
		switch k & typ.MaskElem {
		case typ.KindArr, typ.KindMap:
			mask |= k << shift
			continue
		}
		el.Kind |= k & typ.FlagOpt
		el.Kind = (el.Kind << shift) | mask
		return el, nil
	}
}
