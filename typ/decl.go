package typ

var (
	Void = Type{Kind: KindVoid}
	Any  = Type{Kind: KindAny}
	Typ  = Type{Kind: KindTyp}

	Num  = Type{Kind: BaseNum}
	Bool = Type{Kind: KindBool}
	Int  = Type{Kind: KindInt}
	Real = Type{Kind: KindReal}

	Char = Type{Kind: BaseChar}
	Str  = Type{Kind: KindStr}
	Raw  = Type{Kind: KindRaw}
	UUID = Type{Kind: KindUUID}

	Time = Type{Kind: KindTime}
	Span = Type{Kind: KindSpan}

	List = Type{Kind: BaseList}
	Dict = Type{Kind: BaseDict}

	Infer = Type{Kind: KindRef}

	Sym  = Type{Kind: ExpSym}
	Dyn  = Type{Kind: ExpDyn}
	Tag  = Type{Kind: ExpTag}
	Decl = Type{Kind: ExpDecl}
)

func Opt(t Type) Type     { return Type{t.Kind | FlagOpt, t.Info} }
func Arr(t Type) Type     { return Type{KindArr, &Info{Params: []Param{{Type: t}}}} }
func Map(t Type) Type     { return Type{KindMap, &Info{Params: []Param{{Type: t}}}} }
func Obj(fs []Param) Type { return Type{KindObj, &Info{Params: fs}} }

func Ref(name string) Type  { return Type{KindRef, &Info{Ref: name}} }
func Flag(name string) Type { return Type{KindFlag, &Info{Ref: name}} }
func Enum(name string) Type { return Type{KindEnum, &Info{Ref: name}} }
func Rec(n string) Type     { return Type{KindRec, &Info{Ref: n}} }

func Var(id uint64, opts ...Type) Type {
	t := Type{Kind: Kind(id<<SlotSize) | KindVar}
	if len(opts) > 0 {
		ps := make([]Param, 0, len(opts))
		for _, p := range opts {
			ps = append(ps, Param{Type: p})
		}
		t.Info = &Info{Params: ps}
	}
	return t
}

// IsOpt returns whether t is an optional type and not any.
func (t Type) IsOpt() bool {
	return t.Kind&FlagOpt != 0 && t.Kind&MaskRef != 0
}

// Deopt returns the non-optional type of t if t is a optional type and not any,
// otherwise t is returned as is.
func (t Type) Deopt() (_ Type, ok bool) {
	if ok = t.IsOpt(); ok {
		t.Kind &^= FlagOpt
	}
	return t, ok
}

// Next returns the next sub type of t if t is a arr or map type,
// otherwise t is returned as is.
func (t Type) Next() Type {
	switch t.Kind & MaskElem {
	case KindArr, KindMap:
		if t.Info == nil || len(t.Params) == 0 {
			return Any
		}
		return t.Params[0].Type
	}
	return Void
}

// Last returns the last sub type of t if t is a arr or map type,
// otherwise t is returned as is.
func (t Type) Last() Type {
	el := t.Next()
	for el != Void {
		t = el
		el = t.Next()
	}
	return t
}

// Ordered returns whether type t supports ordering.
func (t Type) Ordered() bool {
	if t.Kind&BaseNum != 0 {
		return true
	}
	switch t.Kind & MaskRef {
	case BaseChar, KindStr, KindEnum:
		return true
	}
	return false
}

// Elem returns a generalized element type for container types and void otherwise.
func (t Type) Elem() Type {
	switch t.Kind & MaskElem {
	case KindArr, KindMap:
		return t.Next()
	case BaseList, BaseDict:
		return Any
	case KindObj:
		// TODO consider an attempt to unify field types
		return Any
	}
	return Void
}
