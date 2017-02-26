package main

import (
	"fmt"
	"os"

	"../../gotalog"
)

// This is a bare bones executor for datalog files.
func main() {
	db := gotalog.NewMemDatabase()
	for i, filename := range os.Args {
		if i == 0 {
			continue
		}
		f, err := os.Open(filename)
		defer f.Close()

		if err != nil {
			panic(err)
		}
		results := make([]gotalog.Result, 0)
		commands, errors := gotalog.Scan(f)
		for command := range commands {
			res, err := gotalog.Apply(command, db)
			if err != nil {
				panic(err)
			}
			if res != nil {
				results = append(results, *res)
			}
		}
		select {
		case err := <-errors:
			if err != nil {
				panic(err)
			}
		default:
		}

		fmt.Print(gotalog.ToString(results))
	}

}
