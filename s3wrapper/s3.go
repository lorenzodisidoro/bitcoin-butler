package s3wrapper

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Errors
var (
	ErrNoSuchKey    = errors.New("The specified key does not exist")
	ErrNoSuchBucket = errors.New("The specified bucket does not exist")
)

// S3Wrapper define service S3 fields
type S3Wrapper struct {
	Client     *s3.S3
	Uploader   *s3manager.Uploader
	Downloader *s3manager.Downloader
	Region     string
}

// New create S3 service client
func (s *S3Wrapper) New(region string) error {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return err
	}

	s.Client = s3.New(session)
	s.Uploader = s3manager.NewUploader(session)
	s.Downloader = s3manager.NewDownloader(session)
	s.Region = region

	return nil
}

// CreateBucket create new S3 bucket
// name as the bucket name
func (s *S3Wrapper) CreateBucket(name string) error {
	_, err := s.Client.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(name)})
	if err != nil {
		return err
	}

	err = s.Client.WaitUntilBucketExists(&s3.HeadBucketInput{
		Bucket: aws.String(name),
	})

	return err
}

// DeleteBucket remove S3 bucket
// name as the bucket name
func (s *S3Wrapper) DeleteBucket(name string) error {
	_, err := s.Client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		return err
	}

	err = s.Client.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: aws.String(name),
	})

	return err
}

// UploadObject uploads an object to a bucket
// bucket as the name of bucket, body as the content body of file
func (s *S3Wrapper) UploadObject(bucket, file, body string) error {
	content := []byte(body)
	_, err := s.Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
		Body:   bytes.NewReader(content),
	})

	return err
}

// GetObject uploads an object to a bucket
// bucket as the name of bucket, file as file name
func (s *S3Wrapper) GetObject(bucket, file string) ([][]string, error) {
	out, err := s.Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	})

	if err != nil {
		errStr := err.Error()

		if strings.Index(errStr, "NoSuchBucket") > -1 {
			return nil, ErrNoSuchBucket
		}

		if strings.Index(errStr, "NoSuchKey") > -1 {
			return nil, ErrNoSuchKey
		}

		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(bytes.NewBuffer(bodyBytes))
	record, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return record, err
}
