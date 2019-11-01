package store

import (
	"fmt"
	"strings"
	"sync"

	"github.com/brunotm/replicant/transaction/manager"
)

var (
	registry = sync.Map{}
)

type Supplier func(uri string) (s manager.Store, err error)

func Register(name string, s Supplier) (err error) {
	if _, ok := registry.Load(name); ok {
		return fmt.Errorf("store: %s already registered", name)
	}
	registry.Store(name, s)
	return nil
}

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
