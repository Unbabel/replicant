package tests

import (
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/Unbabel/replicant/store"
	"github.com/Unbabel/replicant/transaction"
)

// Setup function type for creating a store for tests
type Setup func(t *testing.T) (s store.Store)

// Destroy function type for cleaning up after tests
type Destroy func(t *testing.T, s store.Store)

// Run store test suite
func Run(t *testing.T, s Setup, d Destroy) {

	t.Run("StoreHas", StoreHas(t, s, d))
	t.Run("StoreSetGet", StoreSetGet(t, s, d))
	t.Run("StoreDelete", StoreDelete(t, s, d))
	t.Run("StoreIter", StoreIter(t, s, d))
}

// StoreHas test
func StoreHas(t *testing.T, setup Setup, destroy Destroy) func(t *testing.T) {

	return func(t *testing.T) {
		s := setup(t)
		defer destroy(t, s)

		tests := testCases{
			{config: transaction.Config{Name: "txn1"}, wantErr: false},
			{config: transaction.Config{Name: "txn2"}, wantErr: true},
			{config: transaction.Config{Name: "txn3"}, wantErr: false},
			{config: transaction.Config{Name: "txn4"}, wantErr: true},
			{config: transaction.Config{Name: "txn5"}, wantErr: true},
		}

		for _, tt := range tests {
			if !tt.wantErr {
				s.Set(tt.config.Name, tt.config)
			}
		}

		for _, tt := range tests {
			t.Run(tt.config.Name, func(t *testing.T) {

				gotExists, err := s.Has(tt.config.Name)
				if err != nil {
					t.Fatal(err)
				}

				if gotExists == tt.wantErr {
					t.Errorf("Store.Has() = %v, want %v", gotExists, tt.wantErr)
				}
			})
		}
	}
}

// StoreSetGet test
func StoreSetGet(t *testing.T, setup Setup, destroy Destroy) func(t *testing.T) {
	return func(t *testing.T) {
		s := setup(t)
		defer destroy(t, s)

		tests := testCases{
			{config: transaction.Config{Name: "txn1"}, wantErr: false},
			{config: transaction.Config{Name: "txn2"}, wantErr: true},
			{config: transaction.Config{Name: "txn3"}, wantErr: false},
			{config: transaction.Config{Name: "txn4"}, wantErr: true},
			{config: transaction.Config{Name: "txn5"}, wantErr: true},
		}

		for _, tt := range tests {
			if !tt.wantErr {
				s.Set(tt.config.Name, tt.config)
			}
		}

		for _, tt := range tests {
			t.Run(tt.config.Name, func(t *testing.T) {
				gotConfig, err := s.Get(tt.config.Name)

				if (err != nil) != tt.wantErr {
					t.Errorf("Store.Get() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if !tt.wantErr && !reflect.DeepEqual(gotConfig, tt.config) {
					t.Errorf("Store.Get() = %v, want %v", gotConfig, tt.config)
				}
			})
		}
	}
}

// StoreDelete type
func StoreDelete(t *testing.T, setup Setup, destroy Destroy) func(t *testing.T) {
	return func(t *testing.T) {
		s := setup(t)
		defer destroy(t, s)

		tests := testCases{
			{config: transaction.Config{Name: "txn1"}, wantErr: false},
			{config: transaction.Config{Name: "txn2"}, wantErr: true},
			{config: transaction.Config{Name: "txn3"}, wantErr: false},
			{config: transaction.Config{Name: "txn4"}, wantErr: true},
			{config: transaction.Config{Name: "txn5"}, wantErr: true},
		}

		for _, tt := range tests {
			if !tt.wantErr {
				s.Set(tt.config.Name, tt.config)
			}
		}

		for _, tt := range tests {
			t.Run(tt.config.Name, func(t *testing.T) {
				if err := s.Delete(tt.config.Name); (err != nil) != tt.wantErr {
					t.Errorf("Store.Delete() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	}
}

// StoreIter type
func StoreIter(t *testing.T, setup Setup, destroy Destroy) func(t *testing.T) {
	return func(t *testing.T) {
		s := setup(t)
		defer destroy(t, s)

		tests := testCases{
			{config: transaction.Config{Name: "txn1"}, wantErr: false},
			{config: transaction.Config{Name: "txn2"}, wantErr: false},
			{config: transaction.Config{Name: "txn3"}, wantErr: false},
			{config: transaction.Config{Name: "txn4"}, wantErr: false},
			{config: transaction.Config{Name: "txn5"}, wantErr: false},
		}

		for _, tt := range tests {
			if !tt.wantErr {
				s.Set(tt.config.Name, tt.config)
			}
		}

		var counter int32

		cb := func(n string, c transaction.Config) bool {
			if (n != "") && (n == c.Name) {
				atomic.AddInt32(&counter, 1)
				t.Log(n)
			}
			return true
		}

		if err := s.Iter(cb); err != nil {
			t.Errorf("Store.Iter() error = %v", err)
		}

		if len(tests) != int(counter) {
			t.Errorf("expected %d iterations, got %d", len(tests), counter)
		}
	}
}

type testCases []struct {
	config  transaction.Config
	wantErr bool
}
