package main

import (
	"os"

	"fmt"

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
		if err != nil {
			panic(err)
		}
		cmds, err := gotalog.Parse(f)
		if err != nil {
			panic(err)
		}
		results, err := gotalog.ApplyAll(cmds, db)
		if err != nil {
			panic(err)
		}
		fmt.Print(gotalog.ToString(results))
	}

}
