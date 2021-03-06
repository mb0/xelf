package exp

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

var SkipTraverse = cor.StrError("skip traverse")

type Visitor interface {
	VisitLit(*Atom) error
	VisitType(*Atom) error
	VisitSym(*Sym) error
	EnterNamed(*Tag) error
	LeaveNamed(*Tag) error
	EnterDyn(*Dyn) error
	LeaveDyn(*Dyn) error
	EnterCall(*Call) error
	LeaveCall(*Call) error
}

// Ghost is a no-op visitor, that visits each element, but does not act on any.
type Ghost struct{}

func (Ghost) VisitLit(*Atom) error  { return nil }
func (Ghost) VisitType(*Atom) error { return nil }
func (Ghost) VisitSym(*Sym) error   { return nil }
func (Ghost) EnterNamed(*Tag) error { return nil }
func (Ghost) LeaveNamed(*Tag) error { return nil }
func (Ghost) EnterDyn(*Dyn) error   { return nil }
func (Ghost) LeaveDyn(*Dyn) error   { return nil }
func (Ghost) EnterCall(*Call) error { return nil }
func (Ghost) LeaveCall(*Call) error { return nil }

func Traverse(v Visitor, els ...El) error {
	for _, el := range els {
		err := el.Traverse(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (x *Atom) Traverse(v Visitor) error {
	if x.Typ() == typ.Typ {
		return v.VisitType(x)
	}
	return v.VisitLit(x)
}

func (x *Sym) Traverse(v Visitor) error {
	return v.VisitSym(x)
}

func (x *Tag) Traverse(v Visitor) error {
	err := v.EnterNamed(x)
	if err != nil {
		return muteSkip(err)
	}
	if x.El != nil {
		err = x.El.Traverse(v)
		if err != nil {
			return muteSkip(err)
		}
	}
	return v.LeaveNamed(x)
}

func (x *Dyn) Traverse(v Visitor) error {
	if x == nil {
		return nil
	}
	err := v.EnterDyn(x)
	if err != nil {
		return muteSkip(err)
	}
	err = Traverse(v, x.Els...)
	if err != nil {
		return err
	}
	return v.LeaveDyn(x)
}

func (x *Call) Traverse(v Visitor) error {
	err := v.EnterCall(x)
	if err != nil {
		return muteSkip(err)
	}
	for _, g := range x.Groups {
		err = Traverse(v, g...)
	}
	if err != nil {
		return err
	}
	return v.LeaveCall(x)
}

func muteSkip(err error) error {
	if err == SkipTraverse {
		return nil
	}
	return err
}
