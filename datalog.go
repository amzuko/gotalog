package gotalog

import "strconv"

func (t Term) getID() string {
	if t.isConstant {
		return "c" + t.value
	}
	return "v" + t.value
}

func makeVar(id string) Term {
	return Term{
		isConstant: false,
		value:      id,
	}
}

// Not threadsafe. TODO.
var globalFreshVarState = 0

func makeFreshVar() Term {
	id := strconv.Itoa(globalFreshVarState)
	globalFreshVarState = globalFreshVarState + 1
	return makeVar(id)
}

func makeConst(value string) Term {
	return Term{
		isConstant: true,
		value:      value,
	}
}

type envirionment map[string]Term

// Predicate has name, arity, and optionally
// a function implementing a primitive
type predicate struct {
	Name      string
	Arity     int
	clauses   func() []*clause
	primitive func(literal, *subgoal) []literal
	id        string
}

func predicateID(name string, arity int) string {
	return name + "/" + strconv.Itoa(arity)
}

type literal struct {
	pred  *predicate
	terms []Term
}

func prefixLength(s string) string {
	return strconv.Itoa(len(s)) + s
}

// TODO:cache
func (l *literal) getID() string {
	s := l.pred.id
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
	mapping := make(map[Term]string)
	tag := prefixLength(l.pred.id)
	for i, t := range l.terms {
		tag = tag + prefixLength(t.getTag(i, mapping))
	}
	return tag
}

func (t Term) getTag(i int, mapping map[Term]string) string {
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
	newTerms := make([]Term, len(l.terms))
	for i, t := range l.terms {
		newTerms[i] = t.substitute(env)
	}
	return literal{
		pred:  l.pred,
		terms: newTerms,
	}
}

func (t Term) substitute(env envirionment) Term {
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
func (t Term) shuffle(env envirionment) {
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
	if l.pred.id != other.pred.id {
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

func (t Term) chase(env envirionment) Term {
	if t.isConstant {
		return t
	}
	if tNext, ok := env[t.value]; ok {
		return tNext
	}
	return t
}

func (t Term) unify(other Term, env envirionment) envirionment {
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

func isIn(t Term, l literal) bool {
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
func substituteInClause(c *clause, env envirionment) *clause {
	if len(env) == 0 {
		return c
	}
	newBody := make([]literal, len(c.body))
	for i, l := range c.body {
		newBody[i] = substitute(l, env)
	}
	return &clause{
		head: substitute(c.head, env),
		body: newBody,
	}
}

func renameClause(c *clause) *clause {
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
func isSafe(c *clause) bool {
	for _, t := range c.head.terms {
		if !t.isSafe(c) {
			return false
		}
	}
	return true
}

func (t Term) isSafe(c *clause) bool {
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

type goals map[string]*subgoal

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

type waiter struct {
	c    *clause
	goal *subgoal
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
	if sg, ok := g[selected.getTag()]; ok {
		sg.waiters = append(sg.waiters, waiter{goal: subgoal, c: c})
		for _, fact := range sg.facts {
			resolvent := resolve(c, fact)
			if resolvent != nil {
				g.addClause(subgoal, resolvent)
			}
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

func (g goals) search(sg *subgoal) error {
	l := sg.literal
	if l.pred.primitive != nil {
		l.pred.primitive(l, sg)
	}

	clauses := l.pred.clauses()
	for _, c := range clauses {
		renamed := renameClause(c)
		env := unify(l, renamed.head)
		if env != nil {
			substituted := substituteInClause(renamed, env)
			g.addClause(sg, substituted)
		}
	}
	return nil
}

func ask(l literal) Result {
	subgoals := goals{}
	sg := newSubGoal(l)
	subgoals.merge(sg)
	subgoals.search(sg)

	if len(sg.facts) > 0 {
		answers := make([][]Term, 0)
		for _, l := range sg.facts {
			answers = append(answers, l.terms)
		}
		return Result{
			Name:    l.pred.Name,
			Arity:   l.pred.Arity,
			Answers: answers,
		}
	}
	return Result{}
}
