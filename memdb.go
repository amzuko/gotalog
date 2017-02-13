package gotalog

import "fmt"

type memClauseStore map[string]*clause

func (mem memClauseStore) add(c *clause) {
	mem[c.getID()] = c
}

func (mem memClauseStore) delete(c *clause) {
	delete(mem, c.getID())
}

func (mem memClauseStore) size() int {
	return len(mem)
}

func (mem memClauseStore) clauses() []*clause {
	clauses := make([]*clause, len(mem))
	i := 0
	for _, c := range mem {
		clauses[i] = c
		i = i + 1
	}
	return clauses

}

type memDatabase struct {
	predicates map[string]*predicate
	clauses    map[string]memClauseStore
}

// NewMemDatabase constructs a new in-memory database.
func NewMemDatabase() Database {
	return &memDatabase{
		predicates: make(map[string]*predicate),
		clauses:    make(map[string]memClauseStore),
	}
}

// TODO: we need to somehow intern predicates on the basis of string/int identification,
// so that multiple clauses referring to the same predicate can reach the same
// db of clauses.
func (db *memDatabase) newPredicate(n string, a int) *predicate {
	id := predicateID(n, a)
	if existing, ok := db.predicates[id]; ok {
		return existing
	}

	p := &predicate{
		Name:      n,
		Arity:     a,
		primitive: nil,
		id:        id,
	}

	p.clauses = func() []*clause {
		return db.clauses[p.id].clauses()
	}

	db.predicates[p.id] = p
	db.clauses[p.id] = memClauseStore{}
	return p
}

// assertions should only be made for clauses' whose
// predicates originate within the same database.
func (db memDatabase) assert(c *clause) error {
	if !isSafe(c) {
		return fmt.Errorf("cannot assert unsafe clauses")
	}

	pred := c.head.pred
	// Ignore assertions on primitive predicates
	if pred.primitive != nil {
		return fmt.Errorf("cannot assert on primitive predicates")
	}

	db.clauses[pred.id].add(c)
	return nil
}

func (db memDatabase) retract(c *clause) error {
	pred := c.head.pred
	db.clauses[pred.id].delete(c)

	// If a predicate has no clauses associated with it, remove it from the db.
	if db.clauses[pred.id].size() == 0 {
		delete(db.predicates, pred.id)
		delete(db.clauses, pred.id)
	}
	return nil
}
