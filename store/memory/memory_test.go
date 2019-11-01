package memory

import (
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/brunotm/replicant/transaction"
)

func TestNew(t *testing.T) {
	s := New("")
	defer cleanStore(t, s)

	c := transaction.Config{}
	c.Name = "test-txn"
	s.Set(c.Name, c)

	ok, err := s.Has(c.Name)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("new store doesn't store")
	}
}

func TestStore_Has(t *testing.T) {

	tests := testCases{
		{config: transaction.Config{Name: "txn1"}, wantErr: false},
		{config: transaction.Config{Name: "txn2"}, wantErr: true},
		{config: transaction.Config{Name: "txn3"}, wantErr: false},
		{config: transaction.Config{Name: "txn4"}, wantErr: true},
		{config: transaction.Config{Name: "txn5"}, wantErr: true},
	}

	s := initStore(t, tests)
	defer cleanStore(t, s)

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

func TestStore_Set_Get(t *testing.T) {

	tests := testCases{
		{config: transaction.Config{Name: "txn1"}, wantErr: false},
		{config: transaction.Config{Name: "txn2"}, wantErr: true},
		{config: transaction.Config{Name: "txn3"}, wantErr: false},
		{config: transaction.Config{Name: "txn4"}, wantErr: true},
		{config: transaction.Config{Name: "txn5"}, wantErr: true},
	}

	s := initStore(t, tests)
	defer cleanStore(t, s)

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

func TestStore_Delete(t *testing.T) {

	tests := testCases{
		{config: transaction.Config{Name: "txn1"}, wantErr: false},
		{config: transaction.Config{Name: "txn2"}, wantErr: true},
		{config: transaction.Config{Name: "txn3"}, wantErr: false},
		{config: transaction.Config{Name: "txn4"}, wantErr: true},
		{config: transaction.Config{Name: "txn5"}, wantErr: true},
	}

	s := initStore(t, tests)
	defer cleanStore(t, s)

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

func TestStore_Iter(t *testing.T) {

	tests := testCases{
		{config: transaction.Config{Name: "txn1"}, wantErr: false},
		{config: transaction.Config{Name: "txn2"}, wantErr: false},
		{config: transaction.Config{Name: "txn3"}, wantErr: false},
		{config: transaction.Config{Name: "txn4"}, wantErr: false},
		{config: transaction.Config{Name: "txn5"}, wantErr: false},
	}

	s := initStore(t, tests)
	defer cleanStore(t, s)

	for _, tt := range tests {
		if !tt.wantErr {
			s.Set(tt.config.Name, tt.config)
		}
	}

	var counter int32

	cb := func(n string, c transaction.Config) bool {
		if (n != "") && (n == c.Name) {
			atomic.AddInt32(&counter, 1)
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

type testCases []struct {
	config  transaction.Config
	wantErr bool
}

func initStore(t *testing.T, tc testCases) *Store {
	s := New("")

	for _, c := range tc {
		if !c.wantErr {
			s.Set(c.config.Name, c.config)
		}
	}

	return s
}

func cleanStore(t *testing.T, s *Store) {
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

}
