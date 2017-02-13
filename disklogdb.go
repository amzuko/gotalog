package gotalog

import "io"

type disklogdb struct {
	w       io.Writer
	backing Database
}

func (db *disklogdb) newPredicate(n string, a int) *predicate {
	return db.backing.newPredicate(n, a)
}

func (db *disklogdb) assert(c *clause) error {
	// what happens if one fails and one succeeds?
	err := db.backing.assert(c)
	if err != nil {
		return err
	}
	return writeClause(db.w, c, assert)
}

func (db *disklogdb) retract(c *clause) error {
	err := db.backing.retract(c)
	if err != nil {
		return err
	}
	return writeClause(db.w, c, retract)
}

// NewDiskLogDB returns a database initialized from an io.ReadWritter. All assertions
// and retractions on this databased will be persisted in the log.
func NewDiskLogDB(rw io.ReadWriter, backing Database) (Database, error) {
	ch := make(chan DatalogCommand, 1000)
	go func() {
		for c := range ch {
			_, err := Apply(c, backing)
			if err != nil {
				// TODO
			}
		}
	}()
	commands, errors := Scan(rw)
	for command := range commands {
		_, err := Apply(command, backing)
		if err != nil {
			return nil, err
		}
	}
	select {
	case err := <-errors:
		if err != nil {
			return nil, err
		}
	}
	return &disklogdb{w: rw, backing: backing}, nil
}
