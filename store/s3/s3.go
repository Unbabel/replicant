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
	data       *s3.S3
	bucketName string
	prefix     string
}

// URI in the form of s3://<access>:<secret>/<bucket>/<path>
func New(uri string) (*Store, error) {
	//TODO
	var err error
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "s3" {
		return nil, fmt.Errorf("Invalid uri scheme for s3")
	}
	var awsconfig *aws.Config = aws.NewConfig()
	var creds *credentials.Credentials
	var sess *session.Session
	secretkey, haspassword := u.User.Password()
	if haspassword {
		creds = credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     u.User.Username(),
			SecretAccessKey: secretkey,
		})
		awsconfig.WithCredentials(creds)
	}

	if err != nil {
		return nil, err
	}
	if reg, ok := u.Query()["region"]; ok {
		sess, err = session.NewSession(awsconfig, aws.NewConfig().WithCredentials(credentials.NewEnvCredentials()), aws.NewConfig().WithRegion(reg[0]))
	} else {
		sess, err = session.NewSession(awsconfig, aws.NewConfig().WithCredentials(credentials.NewEnvCredentials()))
	}
	if err != nil {
		return nil, err
	}
	svc := s3.New(sess)

	return &Store{data: svc, bucketName: u.Host, prefix: u.Path}, nil
}

func (s *Store) Close() (err error) {
	//TODO
	return nil
}

func (s *Store) Delete(name string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.prefix + "/" + name),
	}
	_, err := s.data.DeleteObject(input)
	if err != nil {
		return err
	}
	return nil
}

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
				return config, err
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
				return false, err
			}
		}
	}
	return true, nil
}

func (s *Store) Iter(callback func(name string, config transaction.Config) (proceed bool)) (err error) {
	//TODO
	return nil
}

func (s *Store) Set(name string, config transaction.Config) (err error) {
	//TODO
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
		return err
	}
	return nil
}
