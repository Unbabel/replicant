package s3

import (
	"bytes"
	"io/ioutil"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	errNoSuchKey    = awserr.New(s3.ErrCodeNoSuchKey, "key not found", nil)
	errNoSuchBucket = awserr.New(s3.ErrCodeNoSuchBucket, "bucket not found", nil)
	errBucketExists = awserr.New(s3.ErrCodeBucketAlreadyExists, "bucket already exists", nil)
)

type bucket map[string][]byte

var _ s3i = (*mockS3)(nil)

// mockS3 is a S3 mock for the store implementation
type mockS3 struct {
	mtx  sync.RWMutex
	data map[string]bucket
}

func newMockS3() *mockS3 {
	return &mockS3{
		data: map[string]bucket{},
	}
}

// ListBuckets s3
func (s *mockS3) ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	buckets := []*s3.Bucket{}
	for name := range s.data {
		bucket := s3.Bucket{Name: aws.String(name)}
		buckets = append(buckets, &bucket)
	}

	output := s3.ListBucketsOutput{
		Buckets: buckets,
	}

	return &output, nil
}

// DeleteBucket s3
func (s *mockS3) DeleteBucket(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, exists := s.data[*input.Bucket]; !exists {
		return nil, errNoSuchBucket
	}

	delete(s.data, *input.Bucket)
	return &s3.DeleteBucketOutput{}, nil
}

// CreateBucket s3
func (s *mockS3) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, exists := s.data[*input.Bucket]; exists {
		return nil, errBucketExists
	}

	s.data[*input.Bucket] = bucket{}
	return &s3.CreateBucketOutput{}, nil
}

// ListObjects s3
func (s *mockS3) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	bucket, ok := s.data[*input.Bucket]
	if !ok {
		return nil, errNoSuchBucket
	}

	var keys []string
	for key := range bucket {
		if strings.HasPrefix(key, *input.Prefix) {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)
	contents := []*s3.Object{}
	for _, key := range keys {
		value := bucket[key]
		object := s3.Object{
			Key:  aws.String(key),
			Size: aws.Int64(int64(len(value))),
		}
		contents = append(contents, &object)
	}

	output := s3.ListObjectsOutput{
		Contents:    contents,
		IsTruncated: aws.Bool(false),
	}

	return &output, nil
}

// GetObject s3
func (s *mockS3) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	bucket, ok := s.data[*input.Bucket]
	if !ok {
		return nil, errNoSuchBucket
	}

	object, ok := bucket[*input.Key]
	if !ok {
		return nil, errNoSuchKey
	}

	body := ioutil.NopCloser(bytes.NewReader(object))
	output := s3.GetObjectOutput{Body: body}
	return &output, nil

}

// PutObject s3
func (s *mockS3) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	content, _ := ioutil.ReadAll(input.Body)
	bucket, ok := s.data[*input.Bucket]
	if !ok {
		return nil, errNoSuchBucket
	}

	bucket[*input.Key] = content
	return &s3.PutObjectOutput{}, nil
}

// DeleteObject s3
func (s *mockS3) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	bucket, ok := s.data[*input.Bucket]
	if !ok {
		return nil, errNoSuchBucket
	}

	if _, ok := bucket[*input.Key]; !ok {
		return nil, errNoSuchKey
	}

	delete(bucket, *input.Key)
	return &s3.DeleteObjectOutput{}, nil
}
