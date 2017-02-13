package gotalog

import (
	"fmt"
	"sync"
)

type lockingClauseStore map[string]*clause

func (store lockingClauseStore) clauses() []*clause {
	clauses := make([]*clause, len(store))
	i := 0
	for _, c := range store {
		clauses[i] = c
		i = i + 1
	}
	return clauses
}

type lockingDatabase struct {
	predicates map[string]*predicate
	clauses    map[string]lockingClauseStore
	m          sync.RWMutex
}

// NewLockingDatabase constructs a new in-memory database with simple locking behavior.
func NewLockingDatabase() Database {
	return &lockingDatabase{
		predicates: make(map[string]*predicate),
		clauses:    make(map[string]lockingClauseStore),
	}
}

func (db *lockingDatabase) newPredicate(n string, a int) *predicate {

	id := predicateID(n, a)
	db.m.RLock()
	if existing, ok := db.predicates[id]; ok {
		db.m.RUnlock()
		return existing
	}
	db.m.RUnlock()

	p := &predicate{
		Name:      n,
		Arity:     a,
		primitive: nil,
		id:        id,
	}

	p.clauses = func() []*clause {
		db.m.RLock()
		defer db.m.RUnlock()
		return db.clauses[p.id].clauses()
	}

	db.m.Lock()
	db.predicates[p.id] = p
	db.clauses[p.id] = lockingClauseStore{}
	db.m.Unlock()
	return p
}

// assertions should only be made for clauses' whose
// predicates originate within the same database.
func (db *lockingDatabase) assert(c *clause) error {

	if !isSafe(c) {
		return fmt.Errorf("cannot assert unsafe clauses")
	}

	pred := c.head.pred
	// Ignore assertions on primitive predicates
	if pred.primitive != nil {
		return fmt.Errorf("cannot assert on primitive predicates")
	}

	db.m.Lock()
	db.clauses[pred.id][c.getID()] = c
	db.m.Unlock()
	return nil
}

func (db *lockingDatabase) retract(c *clause) error {
	pred := c.head.pred
	db.m.Lock()
	delete(db.clauses[pred.id], c.getID())

	// If a predicate has no clauses associated with it, remove it from the db.
	if len(db.clauses[pred.id]) == 0 {
		delete(db.predicates, pred.id)
		delete(db.clauses, pred.id)
	}
	defer db.m.Unlock()

	return nil
}
