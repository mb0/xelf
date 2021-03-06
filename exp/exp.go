package exp

import (
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lex"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// El is the common interface of all language elements.
type El interface {
	// WriteBfr writes the element to a bfr.Ctx.
	WriteBfr(*bfr.Ctx) error
	// String returns the xelf representation as string.
	String() string
	// Typ returns the element type.
	Typ() typ.Type
	// Source returns the source position if available.
	Source() lex.Src
	// Travers calls the appropriate visitor methods for this element and its children.
	Traverse(Visitor) error
}

// All the language elements
type (
	// Atom is a literal or type with source source offsets as returned by the parser.
	Atom struct {
		Lit lit.Lit
		lex.Src
	}

	// Sym is an identifier, that refers to a definition.
	Sym struct {
		Name string
		// Type is the resolved type of lit in this context or void.
		Type typ.Type
		// Lit is the resolved literal or nil. Conversion may be required.
		Lit lit.Lit
		lex.Src
	}

	// Dyn is an expression with an undefined specification, that has to be determined.
	Dyn struct {
		Els []El
		lex.Src
	}

	// Tag is a named elements. Its meaning is determined by the parent's specification.
	Tag struct {
		Name string
		El   El
		lex.Src
	}

	// Call is an expression with a defined specification.
	Call struct {
		// Spec is the form or func specification
		Spec *Spec
		Layout
		lex.Src
	}
)

func ResType(el El) typ.Type { t, _ := ResInfo(el); return t }

func ResInfo(el El) (typ.Type, lit.Lit) {
	switch v := el.(type) {
	case *Atom:
		return v.Typ(), v.Lit
	case *Sym:
		return v.Type, v.Lit
	case *Call:
		return v.Res(), nil
	} // case *Dyn, *Named:
	return typ.Void, nil
}

func (x *Atom) Typ() typ.Type {
	if x != nil && x.Lit != nil {
		return x.Lit.Typ()
	}
	return typ.Void
}
func (x *Sym) Typ() typ.Type  { return typ.Sym }
func (x *Dyn) Typ() typ.Type  { return typ.Dyn }
func (x *Call) Typ() typ.Type { return typ.Call }
func (x *Tag) Typ() typ.Type  { return typ.Tag }

func (x *Atom) String() string { return bfr.String(x) }
func (x *Sym) String() string  { return x.Name }
func (x *Dyn) String() string  { return bfr.String(x) }
func (x *Tag) String() string  { return bfr.String(x) }
func (x *Call) String() string { return bfr.String(x) }

func (x *Atom) WriteBfr(b *bfr.Ctx) error { return x.Lit.WriteBfr(b) }
func (x *Sym) WriteBfr(b *bfr.Ctx) error  { return b.Fmt(x.Name) }
func (x *Dyn) WriteBfr(b *bfr.Ctx) error  { return writeExpr(b, "", x.Els) }
func (x *Tag) WriteBfr(b *bfr.Ctx) error {
	switch x.Name {
	case ":", ";":
		b.WriteByte('(')
		b.WriteString(x.Name)
		d, ok := x.El.(*Dyn)
		if ok {
			for i, el := range d.Els {
				if i > 0 {
					b.WriteByte(' ')
				}
				el.WriteBfr(b)
			}
		} else if x.El != nil {
			x.El.WriteBfr(b)
		}
		b.WriteByte(')')
	case "":
		if x.El != nil {
			x.El.WriteBfr(b)
		} else {
			b.WriteString("'';")
		}
	default:
		b.WriteString(x.Name)
		if x.El != nil {
			b.WriteByte(':')
			x.El.WriteBfr(b)
		} else {
			b.WriteByte(';')
		}
	}
	return nil
}
func (x *Call) WriteBfr(b *bfr.Ctx) error {
	name := x.Spec.Ref
	if name == "" {
		name = x.Spec.String()
	}
	return writeExpr(b, name, x.All())
}

func writeExpr(b *bfr.Ctx, name string, args []El) error {
	b.WriteByte('(')
	if name != "" {
		b.WriteString(name)
		if len(args) != 0 && name != ":" && name != ";" {
			b.WriteByte(' ')
		}
	}
	for i, x := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		err := x.WriteBfr(b)
		if err != nil {
			return err
		}
	}
	return b.WriteByte(')')
}
func (x *Sym) Key() string { return cor.Keyed(x.Name) }

// Res returns the result type or void.
func (x *Call) Res() typ.Type {
	if isSig(x.Sig) {
		return x.Sig.Params[len(x.Sig.Params)-1].Type
	}
	return x.Spec.Res()
}
func NewNamed(name string, els ...El) *Tag {
	if len(els) == 0 {
		return &Tag{Name: name, El: nil}
	}
	if len(els) > 1 {
		return &Tag{Name: name, El: &Dyn{Els: els}}
	}
	return &Tag{Name: name, El: els[0]}
}

func (x *Tag) Key() string { return cor.Keyed(x.Name) }

func (x *Tag) Args() []El {
	if x.El == nil {
		return nil
	}
	if d, ok := x.El.(*Dyn); ok {
		return d.Els
	}
	return []El{x.El}
}
func (x *Tag) Arg() El {
	if d, ok := x.El.(*Dyn); ok && len(d.Els) != 0 {
		return d.Els[0]
	}
	return x.El
}
func (x *Tag) Dyn() *Dyn {
	if d, ok := x.El.(*Dyn); ok {
		return d
	}
	return nil
}
