package store

import (
	"fmt"
	"strings"
	"sync"

	"github.com/brunotm/replicant/transaction"
)

var (
	registry = sync.Map{}
	// ErrTransactionNotFound transaction not found
	ErrTransactionNotFound = fmt.Errorf("transaction not found")
)

// Store for transaction configurations
type Store interface {

	// Close the store
	Close() (err error)

	// Has checks if a transaction config exists under the given name
	Has(name string) (exists bool, err error)

	// Get a transaction config from the store
	Get(name string) (config transaction.Config, err error)

	// Set stores the given transaction config
	Set(name string, config transaction.Config) (err error)

	// Delete the transaction config for the given name
	Delete(name string) (err error)

	// Iter iterates the transaction configs applying the callback for the name and config pairs.
	// Returning false causes the iteration to stop.
	Iter(callback func(name string, config transaction.Config) (proceed bool)) (err error)
}

// Supplier for manager.Store
type Supplier func(uri string) (s Store, err error)

// Register registers store suppliers
func Register(name string, s Supplier) (err error) {
	if _, ok := registry.Load(name); ok {
		return fmt.Errorf("store: %s already registered", name)
	}
	registry.Store(name, s)
	return nil
}

// New creates a new store with the registered suppliers from the given URI.
// URI spec: <store>:<arguments>
func New(uri string) (s Store, err error) {
	params := strings.SplitN(uri, ":", 2)
	if len(params) == 0 {
		return nil, fmt.Errorf("store: invalid uri %s", uri)
	}
	name := params[0]
	spi, ok := registry.Load(name)
	if !ok {
		return nil, fmt.Errorf("store: %s not registered", name)
	}

	sp := spi.(Supplier)
	return sp(uri)
}
