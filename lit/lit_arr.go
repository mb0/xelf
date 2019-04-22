package lit

import (
	"reflect"

	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

// MakeList returns a new abstract list literal with the given type and len or an error.
func MakeList(t typ.Type, len int) (*List, error) {
	return MakeListCap(t, len, len)
}

// MakeListCap returns a new abstract list literal with the given type, len and cap or an error.
func MakeListCap(t typ.Type, len, cap int) (*List, error) {
	if t.Kind&typ.MaskElem != typ.KindList {
		return nil, typ.ErrInvalid
	}
	res := List{t.Elem(), make(Idxr, len, cap)}
	for i := range res.Idxr {
		res.Idxr[i] = Null(res.Elem)
	}
	return &res, nil
}

type (
	List struct {
		Elem typ.Type
		Idxr
	}
	proxyList struct{ proxy }
)

func (a List) Typ() typ.Type     { return typ.List(a.Elem) }
func (a List) Element() typ.Type { return a.Elem }
func (a List) SetIdx(i int, el Lit) (_ Indexer, err error) {
	if el == nil {
		el = Null(a.Elem)
	} else {
		el, err = Convert(el, a.Elem, 0)
		if err != nil {
			return a, err
		}
	}
	_, err = a.Idxr.SetIdx(i, el)
	return a, err
}

func (a List) Append(ls ...Lit) (Appender, error) {
	for _, e := range ls {
		e, err := Convert(e, a.Elem, 0)
		if err != nil {
			return nil, err
		}
		a.Idxr = append(a.Idxr, e)
	}
	return a, nil
}

func (p *proxyList) Assign(l Lit) error {
	if l == nil || !p.typ.Equal(l.Typ()) {
		return cor.Errorf("%q not assignable to %q", l.Typ(), p.typ)
	}
	b, ok := l.(Indexer)
	if !ok || b.IsZero() { // a nil list?
		if p.val.CanAddr() {
			p.val.Set(reflect.Zero(p.val.Type()))
		} else {
			p.val = reflect.New(p.val.Type().Elem())
		}
		return nil
	}
	v, ok := p.elem(reflect.Slice)
	if !ok {
		return ErrNotSlice
	}
	v = v.Slice(0, 0)
	err := b.IterIdx(func(i int, e Lit) error {
		fp := reflect.New(v.Type().Elem())
		fl, err := ProxyValue(fp)
		if err != nil {
			return err
		}
		err = fl.Assign(e)
		if err != nil {
			return err
		}
		v = reflect.Append(v, fp.Elem())
		return nil
	})
	if err != nil {
		return err
	}
	pv := p.val.Elem()
	if pv.Kind() == reflect.Ptr {
		pv = pv.Elem()
	}
	pv.Set(v)
	return nil
}

func (p *proxyList) Append(ls ...Lit) (Appender, error) {
	v, ok := p.elem(reflect.Slice)
	if !ok {
		return nil, ErrNotSlice
	}
	rt := v.Type().Elem()
	for _, e := range ls {
		fp := reflect.New(rt)
		err := AssignToValue(e, fp)
		if err != nil {
			return nil, err
		}
		v = reflect.Append(v, fp.Elem())
	}
	res := *p
	res.val = reflect.New(v.Type())
	res.val.Set(v)
	return &res, nil
}

func (p *proxyList) Element() typ.Type { return p.typ.Elem() }
func (p *proxyList) Len() int {
	if v, ok := p.elem(reflect.Slice); ok {
		return v.Len()
	}
	return 0
}
func (p *proxyList) IsZero() bool { return p.Len() == 0 }
func (p *proxyList) Idx(i int) (Lit, error) {
	if v, ok := p.elem(reflect.Slice); ok {
		if i >= 0 && i < v.Len() {
			return ProxyValue(v.Index(i).Addr())
		}
	}
	return nil, ErrIdxBounds
}
func (p *proxyList) SetIdx(i int, l Lit) (Indexer, error) {
	if v, ok := p.elem(reflect.Slice); ok {
		if i >= 0 && i < v.Len() {
			return p, AssignToValue(l, v.Index(i).Addr())
		}
	}
	return p, ErrIdxBounds
}
func (p *proxyList) IterIdx(it func(int, Lit) error) error {
	if v, ok := p.elem(reflect.Slice); ok {
		for i, n := 0, v.Len(); i < n; i++ {
			el, err := ProxyValue(v.Index(i).Addr())
			if err != nil {
				return err
			}
			err = it(i, el)
			if err != nil {
				if err == BreakIter {
					return nil
				}
				return err
			}
		}
	}
	return nil
}

func (p *proxyList) String() string               { return bfr.String(p) }
func (p *proxyList) MarshalJSON() ([]byte, error) { return bfr.JSON(p) }
func (p *proxyList) WriteBfr(b *bfr.Ctx) error {
	b.WriteByte('[')
	err := p.IterIdx(func(i int, el Lit) error {
		if i > 0 {
			writeSep(b)
		}
		return writeLit(b, el)
	})
	if err != nil {
		return err
	}
	return b.WriteByte(']')
}

var _, _ Appender = &List{}, &proxyList{}
