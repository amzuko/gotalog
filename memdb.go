package gotalog

import "fmt"

type memClauseStore map[string]*clause

func (mem memClauseStore) add(c *clause) error {
	mem[c.getID()] = c
	return nil
}

func (mem memClauseStore) delete(c *clause) error {
	delete(mem, c.getID())
	return nil
}

func (mem memClauseStore) size() (int, error) {
	return len(mem), nil
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

func (db *memDatabase) insert(pred *predicate) {
	db.predicates[pred.id] = pred
}

func (db *memDatabase) remove(pred predicate) predicate {
	delete(db.predicates, pred.id)
	return pred
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

	return db.clauses[pred.id].add(c)
}

func (db memDatabase) retract(c *clause) error {
	pred := c.head.pred
	err := db.clauses[pred.id].delete(c)
	if err != nil {
		// This leads to garbage in the predicate's database.
		return err
	}

	// If a predicate has no clauses associated with it, remove it from the db.
	size, err := db.clauses[pred.id].size()
	if err != nil {
		// Likewise, we end up with garbage if this happens.
		return err
	}

	if size == 0 {
		db.remove(*pred)
	}
	return nil
}
