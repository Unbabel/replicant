package leveldb

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brunotm/replicant/store"
	"github.com/brunotm/replicant/transaction"
	"github.com/brunotm/replicant/transaction/manager"
	"github.com/syndtr/goleveldb/leveldb"
)

var _ manager.Store = (*Store)(nil)

func init() {
	store.Register("leveldb",
		func(uri string) (s manager.Store, err error) {
			return New(uri)
		})
}

// Store is a in memory transaction config store
type Store struct {
	data *leveldb.DB
}

// New creates a new in memory transaction config store
func New(uri string) (s *Store, err error) {
	s = &Store{}

	params := strings.SplitN(uri, ":", 2)
	if len(params) == 0 {
		return nil, fmt.Errorf("store: invalid uri %s", uri)
	}
	path := params[1]

	s.data, err = leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Close the store
func (s *Store) Close() (err error) {
	return s.data.Close()
}

// Has checks if a transaction config exists under the given name
func (s *Store) Has(name string) (exists bool, err error) {
	return s.data.Has([]byte(name), nil)
}

// Get a transaction config from the store
func (s *Store) Get(name string) (config transaction.Config, err error) {
	b, err := s.data.Get([]byte(name), nil)

	switch {
	case err == leveldb.ErrNotFound:
		return config, store.ErrTransactionNotFound
	case err != nil:
		return config, err
	}

	err = json.Unmarshal(b, &config)
	if err != nil {
		return config, err
	}

	return config, nil

}

// Set stores the given transaction config
func (s *Store) Set(name string, config transaction.Config) (err error) {

	b, err := json.Marshal(&config)
	if err != nil {
		return err
	}

	return s.data.Put([]byte(name), b, nil)
}

// Delete the transaction config for the given name
func (s *Store) Delete(name string) (err error) {
	ok, err := s.Has(name)
	if err != nil {
		return err
	}

	if !ok {
		return store.ErrTransactionNotFound
	}

	return s.data.Delete([]byte(name), nil)
}

// Iter iterates the transaction configs applying the callback for the name and config pairs.
// Returning false causes the iteration to stop.
func (s *Store) Iter(callback func(name string, config transaction.Config) (proceed bool)) (err error) {
	iter := s.data.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		var config transaction.Config
		name := string(iter.Key())

		err = json.Unmarshal(iter.Value(), &config)
		if err != nil {
			return err
		}

		if !callback(name, config) {
			return
		}
	}
	return iter.Error()
}
