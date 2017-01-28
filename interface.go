package gotalog

import "fmt"

// Database holds and generates state for asserted facts and rules.
// We're mirroring the original implementation's use of 'database'. Unfortunately,
// this was used to describe a number of different uses for tables mapping From
// some string id to some type. TODO: consider renaming other uses of 'database'
// for clarity.
type Database interface {
	newPredicate(n string, a int) *predicate
	assert(c clause) error
	retract(c clause) error
}

type clauseStore interface {
	set(string, clause) error
	delete(string) error
	size() (int, error)
	iterator() (chan clause, error)
}

// TODO: consider whether this should actually be the public API,
// rather than the assert/search api?
type makeLiteral struct {
	pName string
	terms []term
}
type commandType int

const (
	assert commandType = iota
	query
	retract
)

// DatalogCommand a command to mutate or query a gotalog database.
type DatalogCommand struct {
	head        makeLiteral
	body        []makeLiteral
	commandType commandType
}

func buildLiteral(ml makeLiteral, db Database) literal {
	return literal{
		pred:  db.newPredicate(ml.pName, len(ml.terms)),
		terms: ml.terms,
	}
}

func apply(cmd DatalogCommand, db Database) (*Result, error) {
	head := buildLiteral(cmd.head, db)
	switch cmd.commandType {
	case assert:
		body := make([]literal, len(cmd.body))
		for i, ml := range cmd.body {
			body[i] = buildLiteral(ml, db)
		}
		err := db.assert(clause{
			head: head,
			body: body,
		})
		return nil, err
	case query:
		res, err := ask(head)
		return &res, err
	case retract:
		body := make([]literal, len(cmd.body))
		for i, ml := range cmd.body {
			body[i] = buildLiteral(ml, db)
		}
		db.retract(clause{
			head: head,
			body: body,
		}) // really, no errors can happen?
		return nil, nil
	}
	return nil, fmt.Errorf("bogus command - this should never happen")
}

// Result contain deduced facts that match a query.
type Result struct {
	Name    string
	Arity   int
	Answers [][]term
}

// ApplyAll iterates over a slice of commands, executes each in turn
// on a provided database, and accumulates and then returns results.
func ApplyAll(cmds []DatalogCommand, db Database) (results []Result, err error) {
	for _, cmd := range cmds {
		res, err := apply(cmd, db)
		if err != nil {
			return results, err
		}
		if res != nil {
			results = append(results, *res)
		}
	}
	return
}
