package gotalog

import (
	"io"
)

type disklogdb struct {
	w       io.Writer
	backing Database
}

func (db *disklogdb) newPredicate(n string, a int) *predicate {
	return db.backing.newPredicate(n, a)
}

func (db *disklogdb) assert(c clause) error {
	// what happens if one fails and one succeeds?
	err := db.backing.assert(c)
	if err != nil {
		return err
	}
	return writeClause(db.w, &c, assert)
}

func (db *disklogdb) retract(c clause) error {
	err := db.backing.retract(c)
	if err != nil {
		return err
	}
	return writeClause(db.w, &c, retract)
}

// NewDiskLogDB returns a database initialized from an io.ReadWritter. All assertions
// and retractions on this databased will be written to the log.
func NewDiskLogDB(rw io.ReadWriter, backing Database) (Database, error) {
	commands, err := Parse(rw)
	if err != nil {
		return nil, err
	}
	// Discard results -- this should be empty anyway.
	_, err = ApplyAll(commands, backing)
	if err != nil {
		return nil, err
	}
	return &disklogdb{w: rw, backing: backing}, nil
}
