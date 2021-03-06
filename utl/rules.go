package utl

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

var layoutSig = exp.MustSig("<form _ args; any>")

// ParseTags parses args as tags and sets them to v using rules or returns an error.
func ParseTags(p *exp.Prog, env exp.Env, els []exp.El, v interface{}, rules TagRules) error {
	n, err := GetNode(v)
	if err != nil {
		return err
	}
	lo, err := exp.FormLayout(layoutSig, els)
	if err != nil {
		return err
	}
	return rules.Resolve(p, env, lo.Tags(0), n)
}

type (
	// IdxKeyer returns a key for an unnamed tag at idx.
	IdxKeyer = func(n Node, idx int) string
	// KeyPrepper resolves els and returns a literal for key or an error.
	KeyPrepper = func(p *exp.Prog, env exp.Env, n *exp.Tag) (lit.Lit, error)
	// KeySetter sets l to node with key or returns an error.
	KeySetter = func(n Node, key string, l lit.Lit) error
)

// KeyRule is a configurable helper for assigning tags to nodes.
type KeyRule struct {
	KeyPrepper
	KeySetter
}

// TagRules is a configurable helper for assigning tags to nodes.
type TagRules struct {
	// Rules holds optional per key rules
	Rules map[string]KeyRule
	// IdxKeyer will map unnamed tags to a key, when null unnamed tags result in an error
	IdxKeyer
	// KeyRule holds optional default rules.
	// If neither specific nor default rules are found DynPrepper and PathSetter are used.
	KeyRule
}

// WithOffset return a with an offset keyer.
func (tr TagRules) WithOffset(off int) *TagRules {
	tr.IdxKeyer = OffsetKeyer(off)
	return &tr
}

// Resolve resolves tags using c and env and assigns them to node or returns an error
func (tr *TagRules) Resolve(p *exp.Prog, env exp.Env, tags []*exp.Tag, node Node) (err error) {
	for i, t := range tags {
		err = tr.ResolveTag(p, env, t, i, node)
		if err != nil {
			return cor.Errorf("resolve tag %q %v for %T: %w", t.Name, t.El, node.Typ(), err)
		}
	}
	return nil
}

// ResolveTag resolves tag using c and env and assigns them to node or returns an error
func (tr *TagRules) ResolveTag(p *exp.Prog, env exp.Env, tag *exp.Tag, idx int, node Node) (err error) {
	var key string
	if tag.Name != "" {
		key = tag.Key()
	} else if tr.IdxKeyer != nil {
		key = tr.IdxKeyer(node, idx)
	}
	if key == "" {
		return cor.Errorf("unrecognized tag %s", tag)
	}
	r := tr.Rules[key]
	l, err := tr.prepper(r)(p, env, tag)
	if err != nil {
		return cor.Errorf("prepper %q err: %w", key, err)
	}
	return tr.setter(r)(node, key, l)
}

// ZeroKeyer is an index keyer without offset.
var ZeroKeyer = OffsetKeyer(0)

// OffsetKeyer returns an index keyer that looks up a field at the index plus the offset.
func OffsetKeyer(offset int) IdxKeyer {
	return func(n Node, i int) string {
		f, err := n.Typ().ParamByIdx(i + offset)
		if err != nil {
			return ""
		}
		return f.Key()
	}
}

// ListPrepper resolves args using c and env and returns a list or an error.
func ListPrepper(p *exp.Prog, env exp.Env, n *exp.Tag) (lit.Lit, error) {
	args, err := p.EvalAll(env, n.Args(), typ.Void)
	if err != nil {
		return nil, err
	}
	res := &lit.List{Data: make([]lit.Lit, 0, len(args))}
	for _, arg := range args {
		res.Data = append(res.Data, arg.(*exp.Atom).Lit)
	}
	return res, nil
}

// DynPrepper resolves args using c and env and returns a literal or an error.
// Empty args return a untyped null literal. Multiple args are resolved as dyn expression.
func DynPrepper(p *exp.Prog, env exp.Env, n *exp.Tag) (lit.Lit, error) {
	args := n.Args()
	if len(args) == 0 {
		return lit.Nil, nil
	}
	var el exp.El
	if len(args) == 1 {
		el = args[0]
	} else {
		el = &exp.Dyn{Els: args}
	}
	x, err := p.Eval(env, el, typ.Void)
	if err != nil {
		return nil, err
	}
	l := x.(*exp.Atom).Lit
	// XXX down cast to generic list, cleanup if container conversion works better
	if ll, ok := l.(*lit.List); ok {
		ll.Elem = typ.Void
	}
	return l, nil
}

// PathSetter sets el to n using key as path or returns an error.
func PathSetter(n Node, key string, el lit.Lit) error {
	path, err := lit.ReadPath(key)
	if err != nil {
		return cor.Errorf("read path %s: %w", key, err)
	}
	_, err = lit.SetPath(n, path, el, true)
	if err != nil {
		return cor.Errorf("set path %s: %w", key, err)
	}
	return nil
}

// ExtraMapSetter returns a key setter that tries to add to a node map field with key.
func ExtraMapSetter(mapkey string) KeySetter {
	return func(n Node, key string, el lit.Lit) error {
		err := PathSetter(n, key, el)
		if err != nil && key != mapkey {
			if el == nil || el == lit.Nil {
				el = lit.True
			}
			err = PathSetter(n, mapkey+"."+key, el)
		}
		return err
	}
}

// BitsPrepper returns a key prepper that tries to resolve a bits constant.
func BitsPrepper(consts []typ.Const) KeyPrepper {
	return func(p *exp.Prog, env exp.Env, n *exp.Tag) (lit.Lit, error) {
		l, err := DynPrepper(p, env, n)
		if err != nil {
			return l, err
		}
		k := n.Key()
		for _, b := range consts {
			if k == b.Key() {
				return lit.Int(b.Val), nil
			}
		}
		return nil, cor.Errorf("no constant named %q", k)
		num, ok := l.(lit.Numeric)
		if !ok {
			return nil, cor.Errorf("expect numer for %q got %T", n.Key(), l)
		}
		return lit.Int(num.Num()), nil
	}
}

// BitsSetter returns a key setter that tries to add to a node bits field with key.
func BitsSetter(key string) KeySetter {
	return func(n Node, _ string, el lit.Lit) error {
		f, err := n.Key(key)
		if err != nil {
			return err
		}
		v, ok := f.(lit.Numeric)
		if !ok {
			return cor.Errorf("expect int field for %q got %T", key, f)
		}
		w, ok := el.(lit.Int)
		if !ok {
			return cor.Errorf("expect int lit for %q got %T", key, el)
		}
		_, err = n.SetKey(key, lit.Int(uint64(v.Num())|uint64(w)))
		return err
	}
}

func (a KeyRule) prepper(r KeyRule) KeyPrepper {
	if r.KeyPrepper != nil {
		return r.KeyPrepper
	}
	if a.KeyPrepper != nil {
		return a.KeyPrepper
	}
	return DynPrepper
}
func (a KeyRule) setter(r KeyRule) KeySetter {
	if r.KeySetter != nil {
		return r.KeySetter
	}
	if a.KeySetter != nil {
		return a.KeySetter
	}
	return PathSetter
}
