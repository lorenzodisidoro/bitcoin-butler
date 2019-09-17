package s3wrapper

import (
	"testing"
)

const awsCliIsInstalledLocally = false

func TestNewBucket(t *testing.T) {
	if awsCliIsInstalledLocally {
		bucketName := "testbucketbutler"
		bucketFile := "my-file"
		fileContent := "Hello world"

		s3 := &S3Wrapper{}
		err := s3.New("eu-west-1")
		if err != nil {
			t.Fatal(err)
		}

		err = s3.CreateBucket(bucketName)
		if err != nil {
			t.Fatal(err)
		}

		err = s3.UploadObject(bucketName, bucketFile, fileContent)
		if err != nil {
			t.Fatal(err)
		}

		objectMap, err := s3.GetObject(bucketName, bucketFile)
		if err != nil {
			t.Fatal(err)
		}

		if objectMap[0][0] != fileContent {
			t.Fatal("Content error")
		}

		err = s3.DeleteBucket(bucketName)
		if err != nil {
			t.Fatal(err)
		}

	}
}
