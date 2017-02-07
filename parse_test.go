package gotalog

import "testing"
import "strings"

type testCase struct {
	s            string
	shouldFail   bool
	commandCount int
}

var cases = []testCase{
	{"foo(bar,baz).", false, 1},
	{"foo(bar,baz)", true, 1},
	{"foo(bar,baz)?", false, 1},
	{"foo(bar,X)?", false, 1},
	{"foo(X,bar)?", false, 1},
	{"foo(bar,baz)~", false, 1},
	{"foo(bar,baz) :- quux(bar, baz).", false, 1},
	{"foo(bar,baz) :- quux(bar, baz), woz(bar).", false, 1},
	{"foo(bar).foo(baz).quux(bar,baz).", false, 3},
	{"foo(bar,baz) :- quux(bar, baz).", false, 1},
	{"foo(bar,baz) :- quux(bar, baz), woz(bar)?", true, 1},
	{"foo(X)?", false, 1},
	{"               \t\tfoo(X) :-    baz ( X )   .", false, 1},
	{"", false, 0},
	{"foo(bar,baz). \n", false, 1},
	{`% Transitive closure test from Guo & Gupta

r(X, Y) :- r(X, Z), r(Z, Y).
r(X, Y) :- p(X, Y), q(Y). %other comment
p(a, b).  p(b, d).  p(b, c).
q(b).  q(c).
r(a, Y)?
`, false, 8},
}

func TestParse(t *testing.T) {
	for i, v := range cases {
		cmds, err := Parse(strings.NewReader(v.s))
		if (err != nil) != v.shouldFail {
			t.Errorf("Case %v: Expected success: %v, got error: %v", i, v.shouldFail, err)
		}
		if !v.shouldFail && len(cmds) != v.commandCount {
			t.Errorf("Case %v: wrong number of commands generated. Got %v, expected %v",
				i, len(cmds), v.commandCount)
		}
	}
}
