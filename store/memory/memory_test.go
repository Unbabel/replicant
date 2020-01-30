package memory

import (
	"testing"

	"github.com/Unbabel/replicant/store"
	"github.com/Unbabel/replicant/store/tests"
)

func TestStore(t *testing.T) {
	tests.Run(t, initStore, cleanStore)
}

func initStore(t *testing.T) store.Store {
	s, err := New("")
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func cleanStore(t *testing.T, s store.Store) {
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

}
