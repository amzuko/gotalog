package gotalog

import "testing"

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
