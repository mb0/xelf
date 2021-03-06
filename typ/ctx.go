package typ

import (
	"fmt"

	"github.com/mb0/xelf/cor"
)

// Ctx is used to check and infer type variables.
type Ctx struct {
	binds Binds
	last  uint64
}

// New returns a new type variable for this context.
func (c *Ctx) New() Type {
	c.last++
	return Var(uint64(c.last))
}

// Bind binds type variable v to type t or returns an error.
func (c *Ctx) Bind(v Kind, t Type) error {
	if v&MaskRef != KindVar {
		return cor.Errorf("not a type variable %s", v)
	}
	id := uint64(v >> SlotSize)
	if id == 0 {
		return cor.Errorf("type variable without id %s", v)
	}
	if id > c.last {
		c.last = id
	}
	if c.Contains(t, v) {
		return cor.Errorf("recursive type variable %s", v)
	}
	c.binds = c.binds.Set(v, t)
	return nil
}

// Apply returns t with variables replaced from context.
func (c *Ctx) Apply(t Type) Type {
	t, _ = c.apply(t, nil)
	t, _ = Choose(t)
	return t
}

func (c *Ctx) apply(t Type, hist []Type) (_ Type, isvar bool) {
	t, isvar = c.unvar(t)
	if !t.HasParams() {
		return t, isvar
	}
	for i := 0; i < len(hist); i++ {
		h := hist[len(hist)-1-i]
		if t.Info == h.Info {
			return h, isvar
		}
	}
	var ps []Param
	hist = append(hist, t)
	for i, p := range t.Params {
		pt, ok := c.apply(p.Type, hist)
		if ok && ps == nil {
			ps = make([]Param, i, len(t.Params))
			copy(ps, t.Params)
		}
		if ps != nil {
			p.Type = pt
			ps = append(ps, p)
		}
	}
	if ps != nil {
		n := *t.Info
		n.Params = ps
		return Type{t.Kind, &n}, true
	}
	return t, isvar
}

// Realize returns the finalized type of t or an error.
// The finalized type is independent of this context.
func (c *Ctx) Realize(t Type) (_ Type, err error) { return c.realize(t, nil) }
func (c *Ctx) realize(t Type, hist [][2]Type) (_ Type, err error) {
	t, _ = c.unvar(t)
	if isVar(t) {
		if !t.HasParams() {
			return t, cor.Errorf("immature type %s", t)
		}
		t = Type{KindAlt, t.Info}
	}
	for i := 0; i < len(hist); i++ {
		h := hist[len(hist)-1-i]
		if t.Info == h[0].Info {
			return h[1], nil
		}
	}
	if t.Kind&MaskRef == KindAlt {
		t, err = Choose(t)
	}
	if !t.HasParams() {
		return t, nil
	}
	n := *t.Info
	n.Params = append(([]Param)(nil), t.Params...)
	res := Type{t.Kind, &n}
	hist = append(hist, [2]Type{t, res})
	for i, p := range n.Params {
		pt, err := c.realize(p.Type, hist)
		if err != nil {
			return res, err
		}
		n.Params[i].Type = pt
	}
	return res, nil
}

func (c *Ctx) unvar(t Type) (_ Type, isvar bool) {
	for isVar(t) {
		isvar = true
		if s, ok := c.binds.Get(t.Kind); ok {
			t = s
			continue
		}
		break
	}
	return t, isvar
}

// Inst instantiates type t for this context, replacing all type vars.
func (c *Ctx) Inst(t Type) Type { r, _ := c.inst(t, nil, nil); return r }
func (c *Ctx) inst(t Type, m Binds, hist []Type) (Type, Binds) {
	if isVar(t) {
		r, ok := m.Get(t.Kind)
		if t.Kind == KindVar || !ok {
			r = c.New()
			if t.Info != nil {
				nfo := *t.Info
				r.Info = &nfo
			}
			m = m.Set(t.Kind, r)
		} else if t.HasParams() {
			if r.Info == nil {
				r.Info = &Info{}
			}
			r.Params = append(r.Params, t.Params...)
			return r, m
		}
		return r, m
	} else if t.HasParams() {
		for i := 0; i < len(hist); i++ {
			h := hist[len(hist)-1-i]
			if t.Info == h.Info {
				return h, m
			}
		}
		n := *t.Info
		r := Type{Kind: t.Kind, Info: &n}
		r.Params = make([]Param, 0, len(t.Params))
		hist = append(hist, t)
		for _, p := range t.Params {
			p.Type, m = c.inst(p.Type, m, hist)
			r.Params = append(r.Params, p)
		}
		return r, m
	}
	return t, m
}

// Bound returns vars with all type variables in t, that are bound to this context, appended.
func (c *Ctx) Bound(t Type, vars Vars) Vars {
	if isVar(t) {
		if _, ok := c.binds.Get(t.Kind); ok {
			vars = vars.Add(t.Kind)
		}
	} else if t.HasParams() {
		for _, p := range t.Params {
			vars = c.Bound(p.Type, vars)
		}
	}
	return vars
}

// Free returns vars with all unbound type variables in t appended.
func (c *Ctx) Free(t Type, vars Vars) Vars {
	if isVar(t) {
		if r, ok := c.binds.Get(t.Kind); ok {
			vars = c.Free(r, vars)
			vars = vars.Del(t.Kind)
		} else {
			vars = vars.Add(t.Kind)
		}
	} else if t.HasParams() {
		for _, p := range t.Params {
			vars = c.Free(p.Type, vars)
		}
	}
	return vars
}

// Contains returns whether t contains the type variable v.
func (c *Ctx) Contains(t Type, v Kind) bool {
	for {
		if isVar(t) {
			if t.Kind == v {
				return true
			}
			t, _ = c.binds.Get(t.Kind)
			continue
		}
		if t.HasParams() {
			for _, p := range t.Params {
				if c.Contains(p.Type, v) {
					return true
				}
			}
		}
		return false
	}
}

func (c Ctx) String() string { return fmt.Sprintf("%s", c.binds) }
