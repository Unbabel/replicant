package manager

import (
	"github.com/brunotm/replicant/replicant/transaction"
)

// Store for transaction configurations
type Store interface {

	// Has checks if a transaction config exists under the given name
	Has(name string) (exists bool)

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
