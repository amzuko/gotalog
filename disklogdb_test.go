package gotalog

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestDiskLogDBInterface(t *testing.T) {
	interfaceTest(t, func() Database {
		f, err := ioutil.TempFile("", "logdbtestsInterfaceTests")
		if err != nil {
			panic(err)
		}
		db, err := NewDiskLogDB(f, NewMemDatabase())
		if err != nil {
			panic(err)
		}
		return db
	})
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func testPersistence(t *testing.T, c pCase) {
	commands, err := Parse(strings.NewReader(c.prog))
	panicOnError(err)

	f, err := ioutil.TempFile("", "logdbPersistenceTests")
	panicOnError(err)

	fname := f.Name()

	db, err := NewDiskLogDB(f, NewMemDatabase())
	panicOnError(err)

	results, err := ApplyAll(commands[0:len(commands)-2], db)
	panicOnError(err)

	// Close and reopen the file
	f.Close()
	f, err = os.OpenFile(fname, os.O_RDWR, 0777)
	panicOnError(err)

	db, err = NewDiskLogDB(f, NewMemDatabase())
	panicOnError(err)

	results, err = ApplyAll(commands[len(commands)-2:], db)
	panicOnError(err)

	compareDatalogResult(t, ToString(results), c.expected)

	err = f.Close()
	panicOnError(err)
}

func TestDiskLogPersistence(t *testing.T) {
	for _, c := range programCases {
		testPersistence(t, c)
	}
}
