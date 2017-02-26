package gotalog

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestAsk(t *testing.T) {
	db := NewMemDatabase()

	parent := db.newPredicate("parent", 2)

	abby := makeConst("abby")
	bob := makeConst("bob")
	charlie := makeConst("charlie")

	err := db.assert(&clause{head: literal{parent, []Term{abby, bob}}})
	if err != nil {
		t.Error(err)
	}
	err = db.assert(&clause{head: literal{parent, []Term{abby, charlie}}})
	if err != nil {
		t.Error(err)
	}

	X := makeVar("X")
	results := ask(literal{parent, []Term{abby, X}})

	if len(results.Answers) != 2 {
		t.Fail()
	}

	sibling := db.newPredicate("sibling", 2)
	Y := makeVar("Y")
	Z := makeVar("Z")

	areSiblings := &clause{
		head: literal{sibling, []Term{X, Y}},
		body: []literal{
			{parent, []Term{Z, X}},
			{parent, []Term{Z, Y}},
		},
	}

	err = db.assert(areSiblings)
	if err != nil {
		t.Error(err)
	}

	results = ask(literal{sibling, []Term{X, Y}})
	if err != nil {
		t.Error(err)
	}
	if len(results.Answers) != 4 {
		t.Error("Wrong length of results", results)
	}
}

func parseApplyExecute(t *testing.T, prog string, db Database) string {
	cmds, err := Parse(strings.NewReader(prog))
	if err != nil {
		t.Errorf("Error parsing: %s", err)
		t.Fail()
	}
	results, err := ApplyAll(cmds, db)

	if err != nil {
		t.Error(err)
		t.Fail()
	}
	return ToString(results)
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
	pCase{
		prog: `foo(a,b).
    foo(b,c).
    foo(a,b)~
    foo(X,Y)?`,
		expected: `foo(b, c).
`,
	},
}

func compareDatalogResult(t *testing.T, result string, expected string) {
	if len(result) != len(expected) {
		t.Errorf("Different string lengths. Got:\n%v\nExpected:\n%v\n", result, expected)
	}
	r := bufio.NewReader(strings.NewReader(result))
	for {
		b, _, _ := r.ReadLine()
		if b == nil {
			break
		}
		s := string(b)
		if !strings.Contains(expected, s) {
			t.Errorf("unexpected solution %s", s)
		}
	}
}

func interfaceTest(t *testing.T, newDB func() Database) {
	for _, pCase := range programCases {
		result := parseApplyExecute(t, pCase.prog, newDB())
		compareDatalogResult(t, result, pCase.expected)
	}
	testAllFiles(t, newDB)
}

func TestMemDBInterface(t *testing.T) {
	interfaceTest(t, NewMemDatabase)
}

// These tests come from https://github.com/c-cube/datalog
var files = []string{
	"tests/clique10.pl",
	"tests/clique100.pl",
	"tests/clique200.pl",
	"tests/clique500.pl",
	"tests/clique1000.pl",
}

func checkFile(filename string, newDB func() Database) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	cmds, err := Parse(f)
	if err != nil {
		return err
	}
	db := newDB()
	results, err := ApplyAll(cmds, db)
	if err != nil {
		return err
	}
	if len(results) != 1 {
		return fmt.Errorf("expected a single result")
	}
	if len(results[0].Answers) != 1 {
		return fmt.Errorf("expected a single fact, but got %v", ToString(results))
	}
	return nil
}

func testAllFiles(t *testing.T, newDB func() Database) {
	if testing.Short() {
		t.SkipNow()
	}
	for _, filename := range files {
		err := checkFile(filename, newDB)
		if err != nil {
			t.Error(err)
		}
	}
}

func BenchmarkCliqueMemDB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		checkFile("tests/clique100.pl", NewMemDatabase)
	}
}

func BenchmarkInductionMemDB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		checkFile("tests/induction100.pl", NewMemDatabase)
	}
}

func concurrencyTests(t *testing.T, db Database) {
	query := func(wg *sync.WaitGroup) {
		for i := 0; i < 20; i++ {
			checkFile("tests/clique100.pl", func() Database { return db })
		}
		wg.Done()
	}
	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go query(&wg)
	}
	wg.Wait()
}
