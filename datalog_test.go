package gotalog

import "testing"
import "strings"
import "bufio"

func TestAsk(t *testing.T) {
	db := database{}

	parent := db.newPredicate("parent", 2)

	abby := makeConst("abby")
	bob := makeConst("bob")
	charlie := makeConst("charlie")

	err := db.assert(clause{head: literal{parent, []term{abby, bob}}})
	if err != nil {
		t.Error(err)
	}
	err = db.assert(clause{head: literal{parent, []term{abby, charlie}}})
	if err != nil {
		t.Error(err)
	}

	X := makeVar("X")
	results, err := ask(literal{parent, []term{abby, X}})

	if len(results.answers) != 2 {
		t.Fail()
	}

	sibling := db.newPredicate("sibling", 2)
	Y := makeVar("Y")
	Z := makeVar("Z")

	areSiblings := clause{
		head: literal{sibling, []term{X, Y}},
		body: []literal{
			{parent, []term{Z, X}},
			{parent, []term{Z, Y}},
		},
	}

	err = db.assert(areSiblings)
	if err != nil {
		t.Error(err)
	}

	results, err = ask(literal{sibling, []term{X, Y}})
	if err != nil {
		t.Error(err)
	}
	if len(results.answers) != 4 {
		t.Error("Wrong length of results", results)
	}
}

func parseApplyExecute(t *testing.T, prog string) string {
	cmds, err := parse(strings.NewReader(prog))
	if err != nil {
		t.Errorf("Error parsing: %s", err)
		t.Fail()
	}
	db := database{}
	results, err := applyAll(cmds, &db)

	if err != nil {
		t.Error(err)
		t.Fail()
	}
	str := ""
	for _, result := range results {
		for _, terms := range result.answers {
			str += result.name
			if len(terms) > 0 {
				str += "("
				termStrings := make([]string, len(terms))
				for i, t := range terms {
					termStrings[i] = t.value
				}
				str += strings.Join(termStrings, ", ")
				str += ")"
			}
			str += ".\n"
		}
	}
	return str
}

type pCase struct {
	prog     string
	expected string
}

var programCases = []pCase{
	pCase{
		prog: `% p q test from Chen & Warren
q(X) :- p(X).
q(a).
p(X) :- q(X).
q(X)?

`,
		expected: `q(a).
`,
	},
	pCase{
		prog: `% path test from Chen & Warren
	edge(a, b). edge(b, c). edge(c, d). edge(d, a).
	path(X, Y) :- edge(X, Y).
	path(X, Y) :- edge(X, Z), path(Z, Y).
	path(X, Y) :- path(X, Z), edge(Z, Y).
	path(X, Y)?
	`,
		expected: `path(a, a).
path(a, b).
path(a, c).
path(a, d).
path(b, a).
path(b, b).
path(b, c).
path(b, d).
path(c, a).
path(c, b).
path(c, c).
path(c, d).
path(d, a).
path(d, b).
path(d, c).
path(d, d).
`,
	},
	pCase{
		prog: `% Laps Test
	contains(ca, store, rams_couch, rams).
	contains(rams, fetch, rams_couch, will).
	contains(ca, fetch, Name, Watcher) :-
	    contains(ca, store, Name, Owner),
	    contains(Owner, fetch, Name, Watcher).
	trusted(ca).
	permit(User, Priv, Name) :-
	    contains(Auth, Priv, Name, User),
	    trusted(Auth).
	permit(User, Priv, Name)?
	`,
		expected: `permit(rams, store, rams_couch).
permit(will, fetch, rams_couch).
`,
	},
	pCase{
		prog: `abcdefghi(z123456789,
	z1234567890123456789,
	z123456789012345678901234567890123456789,
	z1234567890123456789012345678901234567890123456789012345678901234567890123456789).

	this_is_a_long_identifier_and_tests_the_scanners_concat_when_read_with_a_small_buffer.
	this_is_a_long_identifier_and_tests_the_scanners_concat_when_read_with_a_small_buffer?`,
		expected: `this_is_a_long_identifier_and_tests_the_scanners_concat_when_read_with_a_small_buffer.
`,
	},
	pCase{
		prog: `% path test from Chen & Warren
edge(a, b). edge(b, c). edge(c, d). edge(d, a).
path(X, Y) :- edge(X, Y).
path(X, Y) :- path(X, Z), edge(Z, Y).
path(X, Y)?`,
		expected: `path(a, a).
path(a, b).
path(a, c).
path(a, d).
path(b, a).
path(b, b).
path(b, c).
path(b, d).
path(c, a).
path(c, b).
path(c, c).
path(c, d).
path(d, a).
path(d, b).
path(d, c).
path(d, d).
`,
	},
	pCase{
		prog: `true.
	true?
	`,
		expected: `true.
`,
	},
}

func TestPrograms(t *testing.T) {
	for _, pCase := range programCases {
		result := parseApplyExecute(t, pCase.prog)
		if len(result) != len(pCase.expected) {
			t.Errorf("Different string lengths. Got:\n%v\nExpected:\n%v\n", result, pCase.expected)
		}
		r := bufio.NewReader(strings.NewReader(result))
		for {
			b, _, _ := r.ReadLine()
			if b == nil {
				break
			}
			s := string(b)
			if !strings.Contains(pCase.expected, s) {
				t.Errorf("unexpected solution %s", s)
			}
		}
	}
}
