package gotalog

import "testing"

func TestLockingDBInterface(t *testing.T) {
	interfaceTest(t, NewLockingDatabase)
}

func BenchmarkCliqueLockingDB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		checkFile("tests/clique100.pl", NewLockingDatabase)
	}
}

func BenchmarkInductionLockingDB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		checkFile("tests/induction100.pl", NewLockingDatabase)
	}
}

func TestLockingConcurrency(t *testing.T) {
	concurrencyTests(t, NewLockingDatabase())
}
