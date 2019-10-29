package memory

import (
	"sync"

	"github.com/brunotm/replicant/transaction"
	"github.com/brunotm/replicant/transaction/manager"
)

var _ manager.Store = (*Store)(nil)

// Store is a in memory transaction config store
type Store struct {
	data sync.Map
}

// New creates a new in memory transaction config store
func New() (s *Store) {
	return &Store{}
}

// Close the store
func (s *Store) Close() (err error) {
	s.data = sync.Map{}
	return nil
}

// Has checks if a transaction config exists under the given name
func (s *Store) Has(name string) (exists bool, err error) {
	_, exists = s.data.Load(name)
	return exists, nil
}

// Get a transaction config from the store
func (s *Store) Get(name string) (config transaction.Config, err error) {
	c, exists := s.data.Load(name)
	if !exists {
		return config, transaction.ErrTransactionNotFound
	}

	config = c.(transaction.Config)
	return config, nil

}

// Set stores the given transaction config
func (s *Store) Set(name string, config transaction.Config) (err error) {
	s.data.Store(name, config)
	return nil
}

// Delete the transaction config for the given name
func (s *Store) Delete(name string) (err error) {
	ok, _ := s.Has(name)
	if !ok {
		return transaction.ErrTransactionNotFound
	}
	s.data.Delete(name)
	return nil
}

// Iter iterates the transaction configs applying the callback for the name and config pairs.
// Returning false causes the iteration to stop.
func (s *Store) Iter(callback func(name string, config transaction.Config) (proceed bool)) (err error) {

	s.data.Range(func(key interface{}, value interface{}) bool {
		name := key.(string)
		config := value.(transaction.Config)

		return callback(name, config)
	})

	return nil
}
