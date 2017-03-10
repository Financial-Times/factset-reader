package main

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var s3TestFolderName = time.Now().Format("2006-01-02")

func TestS3Writer_Gets3ResName(t *testing.T) {
	as := assert.New(t)
	tcs := []struct {
		archive  string
		file     string
		expected string
	}{
		{
			archive:  "edm_premium_full_1532.zip",
			file:     "edm_entity.txt",
			expected: s3TestFolderName + "/edm_entity.txt",
		},
		{
			archive:  "edm_premium_full_1532.zip",
			file:     "edm_entity.txt",
			expected: s3TestFolderName + "/edm_entity.txt",
		},
	}

	wr := S3Writer{}

	for _, tc := range tcs {
		r := wr.getS3ResFilePath(tc.file, tc.archive)
		as.Equal(r, tc.expected)
	}
}

func TestS3Writer_Gets3ResName_NoExtension(t *testing.T) {
	as := assert.New(t)
	tcs := []struct {
		archive  string
		file     string
		expected string
	}{
		{
			archive:  "edm_premium_full_1532",
			file:     "edm_entity.txt",
			expected: s3TestFolderName + "/edm_entity.txt",
		},
	}

	wr := S3Writer{}

	for _, tc := range tcs {
		r := wr.getS3ResFilePath(tc.file, tc.archive)
		as.Equal(tc.expected, r)
	}
}

func TestS3Writer_Gets3ResName_EmptyFilename(t *testing.T) {
	as := assert.New(t)
	tcs := []struct {
		archive  string
		file     string
		expected string
	}{
		{
			archive:  "",
			file:     "",
			expected: "",
		},
	}

	wr := S3Writer{}

	for _, tc := range tcs {
		r := wr.getS3ResFilePath(tc.file, tc.archive)
		as.Equal(r, tc.expected)
	}
}

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
	err := wr.Write(testFolder, "edm_security_entity_map_test.txt", "edm_security_entity_map_test_v1_full_2145.txt")
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
	err := wr.Write(testFolder, "edm_security_entity_map_test.txt", "edm_security_entity_map_test_v1_full_2115.txt")
	as.NotNil(err)
	as.Error(err)
}
