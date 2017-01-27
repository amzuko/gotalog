package gotalog

import "strconv"
import "fmt"
import "strings"

// Term is an inerface implemented by variables
// and constants.
type term struct {
	isConstant bool
	// If term is a constant, value is the constant value.
	// If term is not a constant (ie, is a variable), value contains
	// the variable's id.
	value string
}

func (t term) getID() string {
	if t.isConstant {
		return "c" + t.value
	}
	return "v" + t.value
}

func makeVar(id string) term {
	return term{
		isConstant: false,
		value:      id,
	}
}

// Not threadsafe. TODO.
var globalFreshVarState = 0

func makeFreshVar() term {
	id := strconv.Itoa(globalFreshVarState)
	globalFreshVarState = globalFreshVarState + 1
	return makeVar(id)
}

func makeConst(value string) term {
	return term{
		isConstant: true,
		value:      value,
	}
}

type envirionment map[string]term

// Predicate has name, arity, and optionally
// a function implementing a primitive
type predicate struct {
	Name      string
	Arity     int
	db        map[string]clause
	primitive func(literal, *subgoal) []literal
}

func (p predicate) getID() string {
	return p.Name + "/" + strconv.Itoa(p.Arity)
}

type literal struct {
	pred  *predicate
	terms []term
}

func (l literal) String() string {
	values := make([]string, len(l.terms))
	for i, t := range l.terms {
		values[i] = t.getID()
	}
	return l.pred.getID() + "(" + strings.Join(values, ", ") + ")"
}

func prefixLength(s string) string {
	return strconv.Itoa(len(s)) + s
}

// TODO:cache
func (l *literal) getID() string {
	s := l.pred.getID()
	for _, v := range l.terms {
		s = s + prefixLength(v.value)
	}
	return s
}

// From original implementation comments:
//  Variant tag

//  Two literal's variant tags are the same if there is a one-to-one
//  mapping of variables to variables, such that when the mapping is
//  applied to one literal, the result is a literal that is the same as
//  the other one, when compared using structural equality.  The
//  variant tag is used as a key by the subgoal table.

// TODO:cache in the literal
func (l literal) getTag() string {
	mapping := make(map[term]string)
	tag := prefixLength(l.pred.getID())
	for i, t := range l.terms {
		tag = tag + prefixLength(t.getTag(i, mapping))
	}
	return tag
}

func (t term) getTag(i int, mapping map[term]string) string {
	if t.isConstant {
		return "ct" + t.value
	}
	if _, ok := mapping[t]; !ok {
		mapping[t] = "vt" + strconv.Itoa(i)
	}
	return mapping[t]
}

func substitute(l literal, env envirionment) literal {
	if len(env) == 0 {
		return l
	}
	newTerms := make([]term, len(l.terms))
	for i, t := range l.terms {
		newTerms[i] = t.substitute(env)
	}
	return literal{
		pred:  l.pred,
		terms: newTerms,
	}
}

func (t term) substitute(env envirionment) term {
	if t.isConstant {
		return t
	}
	if v, ok := env[t.value]; ok {
		return v
	}
	return t
}

// Shuffle creates a new envirionement where all
// variables are mapped to freshly generated variables
func shuffle(l literal, env envirionment) envirionment {
	for _, t := range l.terms {
		t.shuffle(env)
	}
	return env
}

// Mutate env
func (t term) shuffle(env envirionment) {
	if !t.isConstant {
		env[t.value] = makeFreshVar()
	}
}

func rename(l literal) literal {
	return substitute(l, shuffle(l, envirionment{}))
}

// Unify that ish!
// From the original docs:
// Unify two literals.  The result is either an environment or nil.
// Nil is returned when the two literals cannot be unified.  When they
// can, applying the substitutions defined by the environment on both
// literals will create two literals that are structurally equal.

func unify(l literal, other literal) envirionment {
	if l.pred.getID() != other.pred.getID() {
		return nil
	}
	env := envirionment{}
	for i, t := range l.terms {
		li := t.chase(env)
		oi := other.terms[i].chase(env)
		if li != oi {
			env = li.unify(oi, env)
			if env == nil {
				return nil
			}
		}
	}
	return env
}

func (t term) chase(env envirionment) term {
	if t.isConstant {
		return t
	}
	if tNext, ok := env[t.value]; ok {
		return tNext
	}
	return t
}

func (t term) unify(other term, env envirionment) envirionment {
	// TODO should move the check for aboslute equality here?
	if t.isConstant && other.isConstant {
		return nil
	} else if other.isConstant {
		env[t.value] = other
	} else {
		env[other.value] = t
	}
	return env
}

func isIn(t term, l literal) bool {
	for _, ti := range l.terms {
		if ti == t {
			return true
		}
	}
	return false
}

// Utilities for handling set of facts.

func isMember(l literal, t map[string]literal) bool {
	_, ok := t[l.getID()]
	return ok
}

func adjoin(l literal, t map[string]literal) {
	t[l.getID()] = l
}

// Clauses

// From the original:
//  A clause has a head literal, and a sequence of literals that form
//  its body.  If there are no literals in its body, the clause is
//  called a fact.  If there is at least one literal in its body, it is
//  called a rule.

//  A clause asserts that its head is true if every literal in its body is
//  true.

type clause struct {
	head literal
	body []literal
}

func (c *clause) getID() string {
	// TODO: cache inside clause
	id := prefixLength(c.head.getID())
	for _, l := range c.body {
		id = id + prefixLength(l.getID())
	}
	return id
}

// Apply a given substitition for each literal.
func substituteInClause(c clause, env envirionment) clause {
	if len(env) == 0 {
		return c
	}
	newBody := make([]literal, len(c.body))
	for i, l := range c.body {
		newBody[i] = substitute(l, env)
	}
	return clause{
		head: substitute(c.head, env),
		body: newBody,
	}
}

func renameClause(c clause) clause {
	env := envirionment{}
	for _, l := range c.body {
		env = shuffle(l, env)
	}
	if len(env) == 0 {
		return c
	}
	return substituteInClause(c, env)
}

// Clause are safe if every variable in their head is in their body.
// This is a key distinction between prolog and datalog, and along with
// the lack of negation, allows us to garuntee that datalog programs
// will terminate.
func isSafe(c clause) bool {
	for _, t := range c.head.terms {
		if !t.isSafe(c) {
			return false
		}
	}
	return true
}

func (t term) isSafe(c clause) bool {
	if t.isConstant {
		return true
	}
	for _, l := range c.body {
		if isIn(t, l) {
			return true
		}
	}
	return false
}

// We're mirroring the original implementation's use of 'database'. Unfortunately,
// this was used to describe a number of different uses for tables mapping From
// some string id to some type. TODO: consider renaming other uses of 'database'
// for clarity.
type database map[string]*predicate

// TODO: we need to somehow intern predicates on the basis of string/int identification,
// so that multiple clauses referring to the same predicate can reach the same
// db of clauses.
func (db database) newPredicate(n string, a int) *predicate {

	p := &predicate{
		Name:      n,
		Arity:     a,
		db:        make(map[string]clause),
		primitive: nil,
	}
	if existing, ok := db[p.getID()]; ok {
		return existing
	}
	db[p.getID()] = p
	return p
}

func (db database) insert(pred *predicate) {
	db[pred.getID()] = pred
}

func (db database) remove(pred predicate) predicate {
	delete(db, pred.getID())
	return pred
}

func (db database) assert(c clause) error {
	if !isSafe(c) {
		return fmt.Errorf("cannot assert unsafe clauses")
	}

	pred := c.head.pred
	// Ignore assertions on primitive predicates
	if pred.primitive != nil {
		return fmt.Errorf("cannot assert on primitive predicates")
	}
	pred.db[c.getID()] = c
	db.insert(pred)
	return nil
}

func (db database) retract(c clause) {
	pred := c.head.pred
	delete(pred.db, c.getID())

	// If a predicate has no clauses associated with it, remove it from the db.
	if len(pred.db) == 0 {
		db.remove(*pred)
	}
}

func (db database) copy() database {
	var newDB database
	for k, v := range db {
		newDB[k] = v
	}
	return newDB
}

func (db database) revert(clone database) {
	db = clone.copy()
}

type goals map[string]*subgoal

func (g goals) String() string {
	if len(g) == 0 {
		return "Empty goals."
	}
	s := "Goals: \n"
	for k, v := range g {
		s = s + k + ": " + v.String() + "\n"
	}
	return s
}

// A subgoal is the item tabled by out solving algorithm.
// A subgoals
type subgoal struct {
	literal literal
	facts   map[string]literal
	waiters []waiter
}

func newSubGoal(l literal) *subgoal {
	return &subgoal{
		literal: l,
		facts:   make(map[string]literal),
	}
}

func (sg subgoal) String() string {
	s := ""
	s = s + sg.literal.String() + "\n"

	if len(sg.facts) > 0 {
		s = s + "Facts:\n"
		values := make([]string, 0)
		for _, v := range sg.facts {
			values = append(values, v.String())
		}
		s = s + strings.Join(values, "\n")
	} else {
		s = s + "No facts."
	}
	s = s + strconv.Itoa(len(sg.waiters)) + " waiters"
	return s
}

type waiter struct {
	c    *clause
	goal *subgoal
}

func (g goals) find(l literal) *subgoal {
	if sg, ok := g[l.getTag()]; ok {
		return sg
	}
	return nil
}

func (g goals) merge(sg *subgoal) {
	g[sg.literal.getTag()] = sg
}

// TODO: probably mroe golang-like to return an error here than nil.
// This is pervasive in the intial port.
func resolve(c *clause, l literal) *clause {
	if len(c.body) == 0 {
		return nil
	}
	env := unify(c.body[0], rename(l))
	if env == nil {
		return nil
	}
	newBody := make([]literal, len(c.body)-1)
	for i, v := range c.body[1:] {
		newBody[i] = substitute(v, env)
	}
	return &clause{
		head: substitute(c.head, env),
		body: newBody,
	}
}

func (g goals) fact(sg *subgoal, l literal) {
	if !isMember(l, sg.facts) {
		adjoin(l, sg.facts)
		for _, w := range sg.waiters {
			resolvent := resolve(w.c, l)
			if resolvent != nil {
				g.addClause(w.goal, resolvent)
			}
		}
	}
}

func (g goals) rule(subgoal *subgoal, c *clause, selected literal) {
	sg := g.find(selected)
	if sg != nil {
		sg.waiters = append(sg.waiters, waiter{goal: subgoal, c: c})
		todo := make([]*clause, 0)
		for _, fact := range sg.facts {
			resolvent := resolve(c, fact)
			if resolvent != nil {
				todo = append(todo, resolvent)
			}
		}
		// TODO: understand why this can't be a part of the above
		// for loop, and write a test that breaks if it changes.
		for _, todoClause := range todo {
			g.addClause(subgoal, todoClause)
		}

	} else {
		sg := newSubGoal(selected)
		sg.waiters = []waiter{waiter{goal: subgoal, c: c}}

		g.merge(sg)
		g.search(sg)
	}
}

func (g goals) addClause(sg *subgoal, c *clause) {
	if len(c.body) == 0 {
		g.fact(sg, c.head)
	} else {
		g.rule(sg, c, c.body[0])
	}
}

func (g goals) search(sg *subgoal) {
	l := sg.literal
	if l.pred.primitive != nil {
		l.pred.primitive(l, sg)
	}
	for _, c := range l.pred.db {
		renamed := renameClause(c)
		env := unify(l, renamed.head)
		if env != nil {
			substituted := substituteInClause(renamed, env)
			g.addClause(sg, &substituted)
		}
	}
}

type result struct {
	name    string
	arity   int
	answers [][]term
}

func ask(l literal) (result, error) {
	subgoals := goals{}
	sg := newSubGoal(l)
	subgoals.merge(sg)
	subgoals.search(sg)

	if len(sg.facts) > 0 {
		answers := make([][]term, 0)
		for _, l := range sg.facts {
			answers = append(answers, l.terms)
		}
		return result{
			name:    l.pred.Name,
			arity:   l.pred.Arity,
			answers: answers,
		}, nil
	}
	return result{}, fmt.Errorf("no results found")
}
