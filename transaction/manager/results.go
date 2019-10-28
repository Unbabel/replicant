package manager

import (
	"errors"
	"sync"

	"github.com/brunotm/replicant/transaction"
)

var errNoResultsFound = errors.New("no results found for transaction")

// Store is a in memory transaction result store
type results struct {
	data sync.Map
}

// Has checks if results exists for a given transaction
func (s *results) Has(name string) (exists bool) {
	_, exists = s.data.Load(name)
	return exists
}

// Get a transaction result from the store
func (s *results) Get(name string) (result transaction.result, err error) {
	r, exists := s.data.Load(name)
	if !exists {
		return result, errNoResultsFound
	}

	result = r.(transaction.Result)
	return result, nil

}

// Set stores the given transaction result
func (s *results) Set(name string, result transaction.Result) (err error) {
	s.data.Store(name, result)
	return nil
}

// Delete the transaction result for the given name
func (s *results) Delete(name string) (err error) {
	if !s.Has(name) {
		return transaction.ErrTransactionNotFound
	}
	s.data.Delete(name)
	return nil
}

// Iter iterates the transaction results applying the callback for the name and result pairs.
// Returning false causes the iteration to stop.
func (s *results) Iter(callback func(name string, result transaction.Result) (proceed bool)) (err error) {

	s.data.Range(func(key interface{}, value interface{}) bool {
		name := key.(string)
		result := value.(transaction.Result)

		return callback(name, result)
	})

	return nil
}
