package std

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

func TestStdFail(t *testing.T) {
	x, err := exp.Read(strings.NewReader(`(fail 'oops')`))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	c := exp.NewProg()
	_, err = c.Eval(Std, x, typ.Void)
	if err == nil {
		t.Fatalf("want err got nothing")
	}
	_, err = c.Resl(Std, x, typ.Void)
	if err != nil {
		t.Fatalf("want no err got %v", err)
	}
}
func TestStdResolveExec(t *testing.T) {
	tests := []struct {
		raw  string
		want lit.Lit
	}{
		{`~any`, typ.Any},
		{`bool`, typ.Bool},
		{`void`, typ.Void},
		{`raw`, typ.Raw},
		{`null`, lit.Nil},
		{`true`, lit.True},
		{`(void anything)`, typ.Void},
		{`(true)`, lit.True},
		{`(bool)`, lit.False},
		{`(bool 1)`, lit.True},
		{`(bool 0)`, lit.False},
		{`(ok 0)`, lit.False},
		{`(raw)`, lit.Raw(nil)},
		{`7`, lit.Num(7)},
		{`(7)`, lit.Num(7)},
		{`()`, typ.Void},
		{`(int 7)`, lit.Int(7)},
		{`(real 7)`, lit.Real(7)},
		{`'abc'`, lit.Char("abc")},
		{`(str)`, lit.Str("")},
		{`(str 'abc')`, lit.Str("abc")},
		{`(raw 'abc')`, lit.Raw("abc")},
		{`(time)`, lit.ZeroTime},
		{`(time null)`, lit.ZeroTime},
		{`(or)`, lit.False},
		{`(or 0)`, lit.False},
		{`(or 1)`, lit.True},
		{`(or 1 (fail))`, lit.True},
		{`(or 0 1)`, lit.True},
		{`(or 1 2 3)`, lit.True},
		{`(and)`, lit.True},
		{`(and 0)`, lit.False},
		{`(and 1)`, lit.True},
		{`(and 1 0)`, lit.False},
		{`(and 0 (fail))`, lit.False},
		{`(and 1 2 3)`, lit.True},
		{`(true 2 3)`, lit.True},
		{`((bool 1) 2 3)`, lit.True},
		{`(not)`, lit.True},
		{`(not 0)`, lit.True},
		{`(not 1)`, lit.False},
		{`(not 0 (fail))`, lit.True},
		{`(not 1 0)`, lit.True},
		{`(not 0 1)`, lit.True},
		{`(not 1 2 3)`, lit.False},
		{`(eq 1 1)`, lit.True},
		{`(eq (int 1) 1)`, lit.True},
		{`(equal (int 1) 1)`, lit.False},
		{`(equal (int 1) (int 1))`, lit.True},
		{`(ne 1 1)`, lit.False},
		{`(ne 0 1)`, lit.True},
		{`(ne 1 1 1)`, lit.False},
		{`(ne 1 1 2)`, lit.True},
		{`(ne 0 1 2)`, lit.True},
		{`(in 2 [1 2 3])`, lit.True},
		{`(ni 2 [1 2 3])`, lit.False},
		{`(in -1 [1 2 3])`, lit.False},
		{`(ni 5 [1 2 3])`, lit.True},
		{`(lt 0 1 2)`, lit.True},
		{`(lt 2 1 0)`, lit.False},
		{`(lt 0 0 2)`, lit.False},
		{`(ge 0 1 2)`, lit.False},
		{`(ge 2 1 0)`, lit.True},
		{`(ge 0 0 2)`, lit.False},
		{`(ge 2 0 0)`, lit.True},
		{`(gt 0 1 2)`, lit.False},
		{`(gt 2 1 0)`, lit.True},
		{`(gt 0 0 2)`, lit.False},
		{`(gt 2 0 0)`, lit.False},
		{`(le 0 1 2)`, lit.True},
		{`(le 2 1 0)`, lit.False},
		{`(le 0 0 2)`, lit.True},
		{`(le 2 0 0)`, lit.False},
		{`(add 1 2)`, lit.Num(3)},
		{`(add 1 2 3)`, lit.Num(6)},
		{`(add -5 2 3)`, lit.Num(0)},
		{`(add (int? 1) 2 3)`, lit.Some{lit.Int(6)}},
		{`(1 2 3)`, lit.Num(6)},
		{`(add (int 1) 2 3)`, lit.Int(6)},
		{`(add (real 1) 2 3)`, lit.Real(6)},
		{`(abs 1)`, lit.Num(1)},
		{`(abs -1)`, lit.Num(1)},
		{`(abs (int -1))`, lit.Int(1)},
		{`(min 1 2 3)`, lit.Num(1)},
		{`(min 3 2 1)`, lit.Num(1)},
		{`(max 1 2 3)`, lit.Num(3)},
		{`(max 3 2 1)`, lit.Num(3)},
		{`(cat 'a' 'b' 'c')`, lit.Str("abc")},
		{`('a' 'b' 'c')`, lit.Str("abc")},
		{`(cat (raw 'a') 'b' 'c')`, lit.Raw("abc")},
		{`(cat [1] [2] [3])`, &lit.List{Data: []lit.Lit{lit.Num(1), lit.Num(2), lit.Num(3)}}},
		{`(apd [] 1 2 3)`, &lit.List{Data: []lit.Lit{lit.Num(1), lit.Num(2), lit.Num(3)}}},
		{`([] 1 2 3)`, &lit.List{Data: []lit.Lit{lit.Num(1), lit.Num(2), lit.Num(3)}}},
		{`(list (list|int 1 2 3))`, &lit.List{Data: []lit.Lit{lit.Int(1), lit.Int(2), lit.Int(3)}}},
		{`(set {} x:2 y:3)`, &lit.Dict{List: []lit.Keyed{
			{"x", lit.Num(2)},
			{"y", lit.Num(3)},
		}}},
		{`({} x:2 y:3)`, &lit.Dict{List: []lit.Keyed{
			{"x", lit.Num(2)},
			{"y", lit.Num(3)},
		}}},
		{`(dict (dict|int x:2 y:3))`, &lit.Dict{List: []lit.Keyed{
			{"x", lit.Int(2)},
			{"y", lit.Int(3)},
		}}},
		{`((real 1) 2 3)`, lit.Real(6)},
		{`(if 1 2)`, lit.Num(2)},
		{`(if 1 2 (fail))`, lit.Num(2)},
		{`(if 1 2 (fail) 3)`, lit.Num(2)},
		{`(if 0 1 2 3)`, lit.Num(3)},
		{`(if 0 1 0 2 3)`, lit.Num(3)},
		{`(if 0 1 0 2)`, lit.Num(0)},
		{`(if 0 (fail) 2)`, lit.Num(2)},
		{`(if 0 (fail))`, lit.Nil},
		{`(if 1 'a')`, lit.Char("a")},
		{`(if 0 'a' 'b')`, lit.Char("b")},
		{`(if 0 'a')`, lit.Char("")},
		{`(let a:(int 1) a)`, lit.Int(1)},
		{`(let a:1 b:2 c:(int (add a b)) c)`, lit.Int(3)},
		{`(let a:1 b:2 c:(add a b) (add a b c))`, lit.Num(6)},
		{`(len 'test')`, lit.Int(4)},
		{`(len [1 2 3])`, lit.Int(3)},
		{`(len {a:1 b:2})`, lit.Int(2)},
		{`(fst [1 2 3 4 5])`, lit.Num(1)},
		{`(lst [1 2 3 4 5])`, lit.Num(5)},
		{`(nth [1 2 3 4 5] 2)`, lit.Num(3)},
		{`(nth [1 2 3 4 5] -3)`, lit.Num(3)},
		{`(fst [1 2 3 4 5] (fn (eq (rem _ 2) 0)))`, lit.Num(2)},
		{`(lst [1 2 3 4 5] (fn (eq (rem _ 2) 0)))`, lit.Num(4)},
		{`(repeat 2 'cool')`, &lit.List{typ.Char, []lit.Lit{
			lit.Char("cool"), lit.Char("cool"),
		}}},
		{`(range 3)`, &lit.List{typ.Int, []lit.Lit{
			lit.Int(0), lit.Int(1), lit.Int(2),
		}}},
		{`(map (range 2) (fn ('row ' (1 _))))`, &lit.List{typ.Char, []lit.Lit{
			lit.Char("row 1"), lit.Char("row 2"),
		}}},
		{`(filter [1 2 3 4 5] (fn (eq (rem _ 2) 0)))`, &lit.List{typ.Any, []lit.Lit{
			lit.Num(2), lit.Num(4),
		}}},
		{`(filter [1 2 3 4 5] (fn (eq (rem _ 2) 1)))`, &lit.List{typ.Any, []lit.Lit{
			lit.Num(1), lit.Num(3), lit.Num(5),
		}}},
		{`(map [1 2 3 4] (fn (mul _ _)))`, &lit.List{Elem: typ.Num,
			Data: []lit.Lit{lit.Num(1), lit.Num(4), lit.Num(9), lit.Num(16)},
		}},
		{`(fold ['alice' 'bob' 'calvin'] (str 'hello')
			(fn (cat _ (if .2 ',') ' ' .1)))`,
			lit.Str("hello alice, bob, calvin"),
		},
		{`(foldr [4 3] [1 2] (fn (apd _ .1)))`, &lit.List{
			Data: []lit.Lit{lit.Num(1), lit.Num(2), lit.Num(3), lit.Num(4)},
		}},
		{`(foldr ['alice' 'bob' 'calvin'] (str 'hello')
			(fn a:str v:str i:int r:str (cat _ ' ' .1 (if .2 ','))))`,
			lit.Str("hello calvin, bob, alice"),
		},
		{`(foldr ['alice' 'bob' 'calvin'] (str 'hello')
			(fn a:str v:str i:int r:str (cat _ ' ' .1 (if .2 ','))))`,
			lit.Str("hello calvin, bob, alice"),
		},
		{`(let a:int @a)`, typ.Int},
		{`(let a:<rec b:int> @a.b)`, typ.Int},
		{`(let a:int b:list|@a @b)`, typ.List(typ.Int)},
		{`(let f:(fn 1) (f))`, lit.Num(1)},
		{`(let f:(fn (int 1)) (f))`, lit.Int(1)},
		{`(let f:(fn res:int 1) (f))`, lit.Int(1)},
		{`(let f:(fn (add _ 1)) (f 1))`, lit.Num(2)},
		{`(let f:(fn (mul _ _)) (f 3))`, lit.Num(9)},
		{`(let f:(fn (int (mul _ _))) (f 3))`, lit.Int(9)},
		{`(let f:(fn b:int r:int (mul _ _)) (f 3))`, lit.Int(9)},
		{`(let sum:(fn n:list|int res:int (fold _ 0 (fn (add _ .1)))) (sum 1 2 3))`,
			lit.Int(6)},
		// TODO fix convert for abstract cont types
		// {`(let sum:(fn (fold _ 0 (fn (add _ .1)))) (sum [1 2 3]))`, lit.Int(6)},
		{`(with 'test' .)`, lit.Char("test")},
		{`(with (<rec a:int> [1]) .a)`, lit.Int(1)},
		{`((fn (eq (add 1 1) 2)))`, lit.True},
		{`(eq true (eq ['a'] ['a']))`, lit.True},
		{`(with [1 2 3 4 5]
			(eq (filter . (fn (equal (rem _ 2) (int 0)))) [2 4])
		)`, lit.True},
		{`(with [1 2 3 4 5]
			(eq (filter . (fn (eq (rem _ 2) (int 0)))) [2 4])
		)`, lit.True},
		{`(with [1 2 3 4 5]
			(eq (fold . [0] (fn (apd _ .1))) [0 1 2 3 4 5])
		)`, lit.True},
		{`(with [1 2 3 4 5] (and
			(eq (fold . (list [0]) (fn (apd _ .1))) (fold . [0] (fn (apd _ .1))))
		))`, lit.True},
		{`(with [1 2 3 4 5] (and
			(eq (foldr . (list [0]) (fn (apd _ .1))) [0 5 4 3 2 1])
			(eq (fold . (list [0]) (fn (apd _ .1))) [0 1 2 3 4 5])
		))`, lit.True},
		{`(with [1 2 3 4 5] (let even:(fn (eq (rem _ 2) 0)) (and
			(eq (len "test") 4)
			(eq (len .) 5)
			(eq (fst .) (nth . 0) 1)
			(eq (lst .) (nth . -1) 5)
			(eq (fst . even) 2)
			(eq (lst . even) 4)
			(eq (nth . 1 even) 4)
			(eq (nth . -2 even) 2)
			(eq (filter . even) [2 4])
			(eq (map . even) [false true false true false])
			(eq (fold . 0 (fn (add _ .1))) 15)
			(eq (foldr . [0] (fn (apd _ .1))) [0 5 4 3 2 1])
			(eq (fold  . [0] (fn (apd _ .1))) [0 1 2 3 4 5])
		)))`, lit.True},
	}
	for _, test := range tests {
		el, err := exp.Read(strings.NewReader(test.raw))
		if err != nil && err != exp.ErrVoid {
			t.Errorf("%s parse err: %v", test.raw, err)
			continue
		}
		p := exp.NewProg()
		r, err := p.Resl(Std, el, typ.Void)
		if err != nil && err != exp.ErrUnres {
			t.Errorf("%s resl err expect ErrUnres, got: %+v\n%v", test.raw, err, p.Unres)
			continue
		}
		r, err = p.Eval(Std, r, typ.Void)
		if err != nil {
			t.Errorf("eval err: %+v\nfor %s\n%v\n", err, test.raw, p.Unres)
			continue
		}
		a, ok := r.(*exp.Atom)
		if !ok {
			t.Errorf("expect atom result got %s", r)
			continue
		}
		if !a.Lit.Typ().Equal(test.want.Typ()) {
			t.Errorf("%s want %s got %s", test.raw, test.want.Typ(), a.Lit.Typ())
		}
		if !reflect.DeepEqual(a.Lit, test.want) {
			t.Errorf("%s want %s got %s", test.raw, test.want, a.Lit)
		}
	}
}

func TestStdResolvePart(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		typ  string
	}{
		{`(or x)`, `(ok x)`, "bool"},
		{`(or 0 x)`, `(ok x)`, "bool"},
		{`(or 1 x)`, `true`, "bool"},
		{`(and x)`, `(ok x)`, "bool"},
		{`(and 0 x)`, `false`, "bool"},
		{`(and 1 x)`, `(ok x)`, "bool"},
		{`(and x v)`, `(and x v)`, "bool"},
		{`(not x)`, `(not x)`, "bool"},
		{`(if 1 x)`, `x`, "num"},
		{`(if 0 1 x)`, `x`, "num"},
		{`(eq 1 x)`, `(eq 1 x)`, "bool"},
		{`(eq 1 x 1)`, `(eq 1 x)`, "bool"},
		{`(eq 1 1 x)`, `(eq 1 x)`, "bool"},
		{`(eq x 1 1)`, `(eq x 1)`, "bool"},
		{`(eq x y 1)`, `(eq x y 1)`, "bool"},
		{`(lt 0 1 x)`, `(lt 1 x)`, "bool"},
		{`(lt 0 x 2)`, `(lt 0 x 2)`, "bool"},
		{`(lt x 1 2)`, `(lt x 1)`, "bool"},
		{`(add x 2 3)`, `(add x 5)`, "num"},
		{`(add 1 x 3)`, `(add 4 x)`, "num"},
		{`(add 1 2 x)`, `(add 3 x)`, "num"},
		{`(sub x 2 3)`, `(sub x 5)`, "num"},
		{`(sub 1 x 3)`, `(sub -2 x)`, "num"},
		{`(sub 1 2 x)`, `(sub -1 x)`, "num"},
		{`(mul x 2 3)`, `(mul x 6)`, "num"},
		{`(mul 6 x 3)`, `(mul 18 x)`, "num"},
		{`(mul 6 2 x)`, `(mul 12 x)`, "num"},
		{`(div x 2 3)`, `(div x 6)`, "num"},
		{`(div 6 x 3)`, `(div 2 x)`, "num"},
		{`(div 6 2 x)`, `(div 3 x)`, "num"},
		{`(1 2 x)`, `(add 3 x)`, "num"},
		{`(bool x)`, `(ok x)`, "bool"},
		{`(abs (bool x))`, `(abs (ok x))`, "bool"},
		{`(int x)`, `(con int x)`, "int"},
		{`(abs (int x))`, `(abs (con int x))`, "int"},
		{`(not (ok x))`, `(not x)`, "bool"},
		{`(not (not x))`, `(ok x)`, "bool"},
		{`(not (not (not x)))`, `(not x)`, "bool"},
		{`(not (not (not (not x))))`, `(ok x)`, "bool"},
		{`(ok (bool x))`, `(ok x)`, "bool"},
		{`(bool (not x))`, `(not x)`, "bool"},
		{`(bool (not (bool x)))`, `(not x)`, "bool"},
		{`(bool (not (bool (not x))))`, `(ok x)`, "bool"},
	}
	env := exp.NewScope(Std)
	env.Def("x", &exp.Def{Type: typ.Num})
	env.Def("y", &exp.Def{Type: typ.Num})
	env.Def("v", &exp.Def{Type: typ.Str})
	for _, test := range tests {
		el, err := exp.Read(strings.NewReader(test.raw))
		if err != nil {
			t.Errorf("%s parse err: %v", test.raw, err)
			continue
		}
		p := exp.NewProg()
		h := p.New()
		r, err := p.Resl(env, el, h)
		if err != nil && err != exp.ErrUnres {
			t.Errorf("%s resl err expect ErrUnres, got: %+v\n%v", test.raw, err, p.Unres)
			continue
		}
		r, err = p.Eval(env, r, h)
		if err != nil && err != exp.ErrUnres {
			t.Errorf("%s eval err expect ErrUnres, got: %+v\n%v", test.raw, err, p.Unres)
			continue
		}
		if got := r.String(); got != test.want {
			t.Errorf("%s want %s got %s", test.raw, test.want, got)
		}
		if got := exp.ResType(r); got.String() != test.typ {
			t.Errorf("%s want %s got %s\n%v", test.raw, test.typ, got, p.Ctx)
		}
	}
}

func TestStdResolve(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		typ  string
	}{
		{`(or 0 1)`, `(or 0 1)`, "bool"},
		{`(and 0 1)`, `(and 0 1)`, "bool"},
		{`(bool 0)`, `(ok 0)`, "bool"},
		{`(not 0)`, `(not 0)`, "bool"},
		{`(if 0 1 2)`, `(if 0 1 2)`, "num"},
		{`(0 1)`, `(add 0 1)`, "num"},
		{`(d 1)`, `(add d 1)`, "int"},
		{`(mul 0 1)`, `(mul 0 1)`, "num"},
		{`(sub 0 1)`, `(sub 0 1)`, "num"},
		{`(div 0 1)`, `(div 0 1)`, "num"},
		{`(rem 0 1)`, `(rem 0 1)`, "int"},
		{`(abs -1)`, `(abs -1)`, "num"},
		{`(neg -1)`, `(neg -1)`, "num"},
		{`(min 0 1)`, `(min 0 1)`, "num"},
		{`(max 0 1)`, `(max 0 1)`, "num"},
		{`(eq 0 1)`, `(eq 0 1)`, "bool"},
		{`(ne 0 1)`, `(ne 0 1)`, "bool"},
		{`(in 0 [1])`, `(in 0 [1])`, "bool"},
		{`(ni 0 [1])`, `(ni 0 [1])`, "bool"},
		{`(lt 0 1)`, `(lt 0 1)`, "bool"},
		{`(cat [0] [1])`, `(cat [0] [1])`, "list"},
		{`([0] 1)`, `(apd [0] 1)`, "list"},
		{`(set {a:0} b:1)`, `(set {a:0} b:1)`, "dict"},
		{`(with {a:0} .a)`, `(with {a:0} .a)`, "num"},
		{`(let a:0 a)`, `(let a:0 a)`, "num"},
		{`(fn (add 1 _))`, `(fn (add 1 _))`, "<func num num>"},
		{`(fn (add d _))`, `(fn (add d _))`, "<func num int>"},
		{`(str '')`, `(con str '')`, "str"},
		{`((fn (add 1 _)) 1)`, `((fn (add 1 _)) 1)`, "num"},
		{`((fn (add d _)) 1)`, `((fn (add d _)) 1)`, "int"},
	}
	env := exp.NewScope(Std)
	env.Def("d", &exp.Def{Type: typ.Int})
	for _, test := range tests {
		x, err := exp.Read(strings.NewReader(test.raw))
		if err != nil {
			t.Errorf("%s parse err: %v", test.raw, err)
			continue
		}
		p := exp.NewProg()
		r, err := p.Resl(env, x, p.New())
		if err != nil && err != exp.ErrUnres {
			t.Errorf("%s resolve err: %v\n%v", test.raw, err, p.Unres)
			continue
		}
		err = p.Realize(r)
		if err != nil {
			t.Errorf("realize err for %s %s: %v", r, callType(r), err)
		}
		if got := r.String(); got != test.want {
			t.Errorf("%s want %s got %s %#v", test.raw, test.want, got, r)
		}
		if got := exp.ResType(r); got.String() != test.typ {
			t.Errorf("%s want %s got %s\n%v", test.raw, test.typ, got, p.Ctx)
		}
	}
}

func callType(el exp.El) typ.Type {
	t := el.Typ()
	switch t.Kind {
	case typ.KindSym:
		s := el.(*exp.Sym)
		return s.Type
	case typ.KindCall:
		x := el.(*exp.Call)
		return x.Sig
	}
	return t
}
