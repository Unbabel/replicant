package s3

import (
	"testing"

	"github.com/Unbabel/replicant/store"
	"github.com/Unbabel/replicant/store/tests"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestStore(t *testing.T) {
	tests.Run(t, initStore, func(t *testing.T, s store.Store) { return })
}

func initStore(t *testing.T) store.Store {

	bucket := "test-bucket"
	s3c := newMockS3()

	if _, err := s3c.CreateBucket(&s3.CreateBucketInput{Bucket: &bucket}); err != nil {
		t.Fatalf("error initializing store: %s", err)
	}

	s := &Store{}
	s.data = s3c
	s.bucketName = bucket
	s.prefix = "replicant"

	return s
}
