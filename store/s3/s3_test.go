package s3

import (
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/Unbabel/replicant/transaction"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestStore_Has(t *testing.T) {

	tests := testCases{
		{config: transaction.Config{Name: "txn1"}, wantErr: false},
		{config: transaction.Config{Name: "txn2"}, wantErr: true},
		{config: transaction.Config{Name: "txn3"}, wantErr: false},
		{config: transaction.Config{Name: "txn4"}, wantErr: true},
		{config: transaction.Config{Name: "txn5"}, wantErr: true},
	}

	s := initStore(t, tests)

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

type testCases []struct {
	config  transaction.Config
	wantErr bool
}

func initStore(t *testing.T, tc testCases) *Store {

	bucket := "test-bucket"
	s3c := newMockS3()

	if _, err := s3c.CreateBucket(&s3.CreateBucketInput{Bucket: &bucket}); err != nil {
		t.Fatalf("error initializing store: %s", err)
	}

	s := &Store{}
	s.data = s3c
	s.bucketName = bucket
	s.prefix = "replicant"

	for _, c := range tc {
		if !c.wantErr {
			s.Set(c.config.Name, c.config)
		}
	}

	return s
}
