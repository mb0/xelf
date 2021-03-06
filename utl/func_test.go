package utl

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lex"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

func TestReflectFunc(t *testing.T) {
	tests := []struct {
		fun   interface{}
		name  string
		names []string
		typ   typ.Type
		err   bool
	}{
		{strings.ToLower, "lower", nil, typ.Func("lower", []typ.Param{
			{Type: typ.Str},
			{Type: typ.Str},
		}), false},
		{strings.Split, "", nil, typ.Func("", []typ.Param{
			{Type: typ.Str},
			{Type: typ.Str},
			{Type: typ.List(typ.Str)},
		}), false},
		{time.Parse, "", nil, typ.Func("", []typ.Param{
			{Type: typ.Str},
			{Type: typ.Str},
			{Type: typ.Time},
		}), true},
		{time.Time.Format, "", []string{"t", "format"}, typ.Func("", []typ.Param{
			{Name: "t", Type: typ.Time},
			{Name: "format", Type: typ.Str},
			{Type: typ.Str},
		}), false},
	}
	for _, test := range tests {
		r, err := ReflectFunc(test.name, test.fun, test.names...)
		if err != nil {
			t.Errorf("reflect for %+v err: %v", test.fun, err)
			continue
		}
		if !test.typ.Equal(r.Type) {
			t.Errorf("for %T want param %s got %s", test.fun, test.typ, r.Type)
		}
		b := r.Impl.(*ReflectBody)
		if test.err != b.err {
			t.Errorf("for %T want err %v got %v", test.fun, test.err, b.err)
		}
	}
}

func TestFuncResolver(t *testing.T) {
	tests := []struct {
		fun   interface{}
		names []string
		args  []exp.El
		want  string
		err   error
	}{
		{strings.ToLower, nil, []exp.El{
			&exp.Atom{Lit: lit.Str("HELLO")},
		}, `'hello'`, nil},
		{time.Time.Format, []string{"t?", "format"}, []exp.El{
			&exp.Tag{Name: ":format", El: &exp.Atom{Lit: lit.Char(`2006-02-01`)}},
		}, `'0001-01-01'`, nil},
		{fmt.Sprintf, nil, []exp.El{
			&exp.Atom{Lit: lit.Str("Hi %s no. %d.")},
			&exp.Atom{Lit: lit.Str("you")},
			&exp.Atom{Lit: lit.Int(9)},
		}, `'Hi you no. 9.'`, nil},
	}
	for _, test := range tests {
		r, err := ReflectFunc("", test.fun, test.names...)
		if err != nil {
			t.Errorf("reflect for %+v err: %v", test.fun, err)
			continue
		}
		p := exp.NewProg()
		c, err := p.NewCall(r, test.args, lex.Src{})
		if err != nil {
			t.Errorf("for %T want err %v", test.fun, err)
			continue
		}
		res, err := r.Eval(p, std.Std, c, typ.Void)
		if err != nil {
			if test.err == nil || test.err.Error() != err.Error() {
				t.Errorf("for %T want err %v got %v", test.fun, test.err, err)
			}
			continue
		}
		if test.want != "" {
			if got := res.String(); test.want != got {
				t.Errorf("for %T want res %s got %v", test.fun, test.want, got)
			}
		}
	}
}
