package gotalog

import (
	"fmt"
	"io"
	"strings"
)

// Database holds and generates state for asserted facts and rules.
// We're mirroring the original implementation's use of 'database'. Unfortunately,
// this was used to describe a number of different uses for tables mapping From
// some string id to some type in the original implementation.
type Database interface {
	newPredicate(n string, a int) *predicate
	assert(c *clause) error
	retract(c *clause) error
}

// Term contains either a variable or a constant.
type Term struct {
	isConstant bool
	// If term is a constant, value is the constant value.
	// If term is not a constant (ie, is a variable), value contains
	// the variable's id.
	value string
}

// LiteralDefinition defines a literal PredicateName(Term0, Term1, ...).
type LiteralDefinition struct {
	PredicateName string
	Terms         []Term
}

// CommandType differentiates different possible datalog commands.
type CommandType int

const (
	// Assert - this fact will be added to a database upon application.
	Assert CommandType = iota
	// Query - this command will return the results of querying a database
	// upon application.
	Query
	// Retract - remove a fact from a database.
	Retract
)

// DatalogCommand a command to mutate or query a gotalog database.
type DatalogCommand struct {
	Head        LiteralDefinition
	Body        []LiteralDefinition
	CommandType CommandType
}

// Parse consumes a reader, producing a slice of datalogCommands.
func Parse(input io.Reader) ([]DatalogCommand, error) {
	s := newScanner(input)

	commands := make([]DatalogCommand, 0)

	for {
		c, finished, err := s.scanOneCommand()
		if err != nil || finished {
			return commands, err
		}
		commands = append(commands, c)
	}
}

// Scan iterates through a io reader, throwing commands into a channel as
// they are read from the reader.
func Scan(input io.Reader) (chan DatalogCommand, chan error) {

	commands := make(chan DatalogCommand, 1000)
	errors := make(chan error)

	s := newScanner(input)

	go func() {
		for {
			c, finished, err := s.scanOneCommand()
			if err != nil {
				errors <- err
				break
			}
			if finished {
				break
			}

			commands <- c
		}
		close(errors)
		close(commands)
	}()
	return commands, errors
}

// Apply applies a single command.
// TODO: do we really need this and ApplyAll?
func Apply(cmd DatalogCommand, db Database) (*Result, error) {
	head := buildLiteral(cmd.Head, db)
	switch cmd.CommandType {
	case Assert:
		body := make([]literal, len(cmd.Body))
		for i, ml := range cmd.Body {
			body[i] = buildLiteral(ml, db)
		}
		err := db.assert(&clause{
			head: head,
			body: body,
		})
		return nil, err
	case Query:
		res := ask(head)
		return &res, nil
	case Retract:
		body := make([]literal, len(cmd.Body))
		for i, ml := range cmd.Body {
			body[i] = buildLiteral(ml, db)
		}
		db.retract(&clause{
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
	Answers [][]Term
}

// ApplyAll iterates over a slice of commands, executes each in turn
// on a provided database, and accumulates and then returns results.
func ApplyAll(cmds []DatalogCommand, db Database) (results []Result, err error) {
	for _, cmd := range cmds {
		res, err := Apply(cmd, db)
		if err != nil {
			return results, err
		}
		if res != nil {
			results = append(results, *res)
		}
	}
	return
}

// ToString reformats results for display.
// Coincidentally, it also generates valid datalog.
func ToString(results []Result) string {
	str := ""
	for _, result := range results {
		for _, terms := range result.Answers {
			str += result.Name
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
