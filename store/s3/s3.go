// Package s3 provides implementation of the storage layer under AWS S3 service.
// This package is a part of the storage packages in replicant.
package s3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/brunotm/replicant/store"
	"github.com/brunotm/replicant/transaction"
	"net/url"
)

var _ store.Store = (*Store)(nil)

func init() {
	store.Register("s3",
		func(uri string) (s store.Store, err error) {
			return New(uri)
		})
}

type Store struct {
	data       *s3.S3 // S3 data source object.
	bucketName string // Name of the bucket to store data.
	prefix     string // Path inside the bucket to use as prefix.
}

// New function creates a new storage object with a subtype of s3.
// receives uri which should be in the form of s3://<access>:<secret>/<bucket>/<prefix>.
// returns a store with subtype s3.
func New(uri string) (*Store, error) {
	var err error
	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if u.Scheme != "s3" {
		return nil, fmt.Errorf("Invalid uri scheme for s3")
	}

	var awsconfig *aws.Config = aws.NewConfig()
	var creds *credentials.Credentials
	var sess *session.Session
	secretKey, hasPassword := u.User.Password()

	if hasPassword {
		creds = credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     u.User.Username(),
			SecretAccessKey: secretKey,
		})
		awsconfig.WithCredentials(creds)
	}

	if err != nil {
		return nil, err
	}

	if reg, ok := u.Query()["region"]; ok {
		sess, err = session.NewSession(awsconfig.WithRegion(reg[0]))
	}

	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	svc := s3.New(sess)

	return &Store{data: svc, bucketName: u.Host, prefix: u.Path}, nil
}

// Close function does nothing as the connection is not persistent.
func (s *Store) Close() (err error) {
	return nil
}

// Delete function deletes the object from the s3 bucket.
// receives a string with the path of the object to be deleted.
// returns an error in case of failing to delete the object.
func (s *Store) Delete(name string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.prefix + "/" + name),
	}
	_, err := s.data.DeleteObject(input)

	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// Get function gets a transaction object from the s3 bucket.
// receives a string with the path to the object.
// returns a transaction configuration and an error in case the object is not found (or any unexpected behaviour).
func (s *Store) Get(name string) (config transaction.Config, err error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.prefix + "/" + name),
	}
	result, err := s.data.GetObject(input)

	if err != nil {

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return config, store.ErrTransactionNotFound
			default:
				return config, fmt.Errorf("%w", err)
			}
		}

	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(result.Body)
	err = json.Unmarshal(buf.Bytes(), &config)

	if err != nil {
		return config, err
	}

	return config, nil
}

// Has function return true or false for either the object exists or not in the s3 bucket.
// receives a string with the path to the object.
// returns a boolean (true if object exists, false otherwise) and an error in case of unexpected behaviour.
func (s *Store) Has(name string) (exists bool, err error) {

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.prefix + "/" + name),
	}
	_, err = s.data.GetObject(input)

	if err != nil {

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return false, nil
			default:
				return false, fmt.Errorf("%w", err)
			}
		}

	}

	return true, nil
}

// TODO
func (s *Store) Iter(callback func(name string, config transaction.Config) (proceed bool)) (err error) {
	//TODO
	return nil
}

// Set function puts a new object or replaces an existing one from the s3 bucket.
// receives a string as the path to identify the object to be put and a transaction configuration which will be the effective object to be written.
// returns an error in case of unexpected behaviour.
func (s *Store) Set(name string, config transaction.Config) (err error) {
	b, err := json.Marshal(&config)

	if err != nil {
		return err
	}

	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(bytes.NewReader(b)),
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.prefix + "/" + name),
	}
	_, err = s.data.PutObject(input)

	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}
