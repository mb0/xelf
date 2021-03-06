package lit

import (
	"bytes"
	"time"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

// Equal returns whether the literals a and b are strictly equal.
func Equal(a, b Lit) bool {
	if res, ok := checkNil(a, b); !ok {
		return res
	}
	if !a.Typ().Equal(b.Typ()) {
		return false
	}
	switch v := a.(type) {
	case typ.Type:
		w, ok := b.(typ.Type)
		return ok && v.Equal(w)
	case Numeric:
		w, ok := b.(Numeric)
		return ok && equalNumer(v, w)
	case Character:
		w, ok := b.(Character)
		return ok && equalCharer(v, w)
	case Indexer:
		w, ok := b.(Indexer)
		return ok && equalIdxer(v, w)
	case Keyer:
		w, ok := b.(Keyer)
		return ok && equalKeyer(v, w)
	}
	return false
}

// Equiv returns whether a and b are equivalent, that is if they are either equal or comparable.
func Equiv(a, b Lit) bool {
	if res, ok := checkNil(a, b); !ok {
		return res
	}
	a, b, ok := comparable(a, b)
	return ok && Equal(a, b)
}

// Less returns whether a is less than b and whether both types are marked as ordered.
func Less(a, b Lit) (res, ok bool) {
	less, _, ok := Comp(a, b)
	return less, ok
}

// Comp returns whether a is less or the same as b and  whether both types marked as ordered.
func Comp(a, b Lit) (less, same, ok bool) {
	if a == nil {
		a = Nil
	}
	if b == nil {
		b = Nil
	}
	a, b, ok = comparable(a, b)
	if !ok || !a.Typ().Ordered() || !b.Typ().Ordered() {
		return false, false, false
	}
	av, vok := a.(valer)
	bv, wok := b.(valer)
	if !vok || !wok {
		return false, false, false
	}
	switch v := av.Val().(type) {
	case bool:
		w, ok := bv.Val().(bool)
		if ok {
			return !v && w, v == w, true
		}
	case int64:
		w, ok := bv.Val().(int64)
		if ok {
			return v < w, v == w, true
		}
	case float64:
		w, ok := bv.Val().(float64)
		if ok {
			return v < w, v == w, true
		}
	case string:
		w, ok := bv.Val().(string)
		if ok {
			return v < w, v == w, true
		}
	case time.Time:
		w, ok := bv.Val().(time.Time)
		if ok {
			return w.After(v), w.Equal(v), true
		}
	case time.Duration:
		w, ok := bv.Val().(time.Duration)
		if ok {
			return v < w, v == w, true
		}
	}
	return false, false, false
}

type valer interface {
	Lit
	Val() interface{}
}

func checkNil(a, b Lit) (eq, ok bool) {
	if a == nil {
		return b == nil, false
	}
	return false, b != nil
}

func comparable(a, b Lit) (v, w Lit, ok bool) {
	cmp := typ.Compare(a.Typ(), b.Typ())
	if cmp < typ.LvlComp {
		cmp = cmp.Mirror()
		if cmp < typ.LvlComp {
			return nil, nil, false
		}
		a, b = b, a
	}
	if cmp != typ.CmpSame {
		c, err := Convert(a, b.Typ(), cmp)
		if err != nil {
			return nil, nil, false
		}
		a = c
	}
	return a, b, true
}

func equalNumer(a, b Numeric) bool {
	switch v := a.Val().(type) {
	case bool:
		w, ok := b.Val().(bool)
		return ok && v == w
	case int64:
		w, ok := b.Val().(int64)
		return ok && v == w
	case float64:
		w, ok := b.Val().(float64)
		return ok && v == w
	case time.Time:
		w, ok := b.Val().(time.Time)
		return ok && v.Equal(w)
	case time.Duration:
		w, ok := b.Val().(time.Duration)
		return ok && v == w
	}
	return false
}

func equalCharer(a, b Character) bool {
	switch v := a.Val().(type) {
	case string:
		w, ok := b.Val().(string)
		return ok && v == w
	case []byte:
		w, ok := b.Val().([]byte)
		return ok && bytes.Equal(v, w)
	case [16]byte:
		w, ok := b.Val().([16]byte)
		return ok && v == w
	case time.Time:
		w, ok := b.Val().(time.Time)
		return ok && v.Equal(w)
	case time.Duration:
		w, ok := b.Val().(time.Duration)
		return ok && v == w
	}
	return false
}

var notEqual = cor.StrError("not equal")

func equalIdxer(a, b Indexer) bool {
	n := a.Len()
	if n != b.Len() {
		return false
	}
	err := a.IterIdx(func(idx int, ae Lit) error {
		be, err := b.Idx(idx)
		if err != nil {
			return err
		}
		if !Equal(ae, be) {
			return notEqual
		}
		return nil
	})
	return err == nil
}

func equalKeyer(a, b Keyer) bool {
	n := a.Len()
	if n != b.Len() {
		return false
	}
	err := a.IterKey(func(key string, ae Lit) error {
		be, err := b.Key(key)
		if err != nil {
			return err
		}
		if !Equal(ae, be) {
			return notEqual
		}
		return nil
	})
	return err == nil
}
