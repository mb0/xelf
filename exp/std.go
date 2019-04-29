package exp

import (
	"log"

	"github.com/mb0/xelf/typ"
)

var StdEnv = Builtin{Core, Std}

var core = make(formMap, 32)
var std = make(formMap, 8)

func Core(sym string) *Spec {
	if f, ok := core[sym]; ok {
		return f
	}
	return nil
}

func Std(sym string) *Spec {
	if f, ok := std[sym]; ok {
		return f
	}
	return nil
}

type CallResolverFunc func(*Ctx, Env, *Call, Type) (El, error)

func (rf CallResolverFunc) ResolveCall(c *Ctx, env Env, e *Call, hint Type) (El, error) {
	return rf(c, env, e, hint)
}

type LayoutResolverFunc = func(*Ctx, Env, *Call, *Layout, Type) (El, error)

func Implement(sig string, resolve bool, r LayoutResolverFunc) *Spec {
	s, err := typ.ParseString(sig)
	if err != nil {
		log.Fatalf("cannot parse spec signature %s: %v", sig, err)
	}
	return &Spec{s, CallResolverFunc(func(c *Ctx, env Env, e *Call, hint Type) (El, error) {
		lo, err := LayoutArgs(e.Spec.Arg(), e.Args)
		if err != nil {
			return nil, err
		}
		if resolve {
			err = lo.Resolve(c, env)
			if err != nil {
				return e, err
			}
		}
		return r(c, env, e, lo, hint)
	})}
}

type formMap map[string]*Spec

func (m formMap) impl(sig string, r LayoutResolverFunc) *Spec {
	f := Implement(sig, false, r)
	m[f.Ref] = f
	return f
}

func (m formMap) implResl(sig string, r LayoutResolverFunc) *Spec {
	f := Implement(sig, true, r)
	m[f.Ref] = f
	return f
}
