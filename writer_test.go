package main

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"io/ioutil"
	"os"
	"time"
)

var s3TestFolderName = time.Now().Format("2006-01-02")

func TestS3Writer_Write(t *testing.T) {
	as := assert.New(t)

	httpS3Client := httpS3ClientMock{
		putObjectMock: func(objectName string, filePath string) (int64, error) {
			file, err := ioutil.ReadFile(filePath)
			if err != nil {
				return 0, err
			}

			err = os.MkdirAll(s3TestFolderName, 0766)
			if err != nil {
				return 0, err
			}
			err = ioutil.WriteFile(objectName, file, 0766)
			if err != nil {
				return 0, err
			}
			f, err := os.Open(objectName)
			if err != nil {
				return 0, err
			}
			b := []byte{}
			n, err := f.Read(b)
			if err != nil {
				return 0, err
			}
			f.Close()
			return int64(n), nil
		},
		bucketExistsMock: func(bucket string) (bool, error) {
			return true, nil
		},
	}
	wr := S3Writer{s3Client: &httpS3Client}
	err := wr.Write(testFolder, "daily.zip")
	as.NoError(err)

	dbFile, err := os.Open(testFolder + "/edm_security_entity_map_test.txt")
	as.NoError(err)
	dbFile.Close()
	err = os.RemoveAll(s3TestFolderName)
}

func TestS3Writer_Write_Error(t *testing.T) {
	as := assert.New(t)

	httpS3Client := httpS3ClientMock{
		putObjectMock: func(objectName string, filePath string) (int64, error) {
			return int64(0), errors.New("Could not connect to Amazaon S3")
		},
		bucketExistsMock: func(bucket string) (bool, error) {
			return true, nil
		},
	}
	wr := S3Writer{s3Client: &httpS3Client}
	err := wr.Write(testFolder, "daily.zip")
	as.NotNil(err)
	as.Error(err)
}
