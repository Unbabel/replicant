package store

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/brunotm/replicant/transaction/manager"
)

var (
	registry = sync.Map{}
	// ErrTransactionNotFound transaction not found
	ErrTransactionNotFound = errors.New("transaction not found")
)

// Supplier for manager.Store
type Supplier func(uri string) (s manager.Store, err error)

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
func New(uri string) (s manager.Store, err error) {
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
