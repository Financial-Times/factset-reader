package main

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
)

const dbFolder = testFolder + "/db"

func TestS3Writer_Gets3ResName(t *testing.T) {
	as := assert.New(t)
	tcs := []struct {
		resName  string
		expected string
	}{
		{
			resName: "edm_premium_full_1532.zip",
			expected: "edm_premium_full_1532" + "_" + time.Now().Format("2006-01-02") + ".zip",
		},
		{
			resName: "edm_premium_full_1532",
			expected: "edm_premium_full_1532" + "_" + time.Now().Format("2006-01-02"),
		},
	}

	s3Writer := s3Writer{}

	for _, tc := range tcs {
		r := s3Writer.gets3ResName(tc.resName)
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
			os.Mkdir(dbFolder, 0766)
			err = ioutil.WriteFile(dbFolder + "/" + objectName, file, 0766)
			if err != nil {
				return 0, err
			}
			f, err := os.Open(dbFolder + "/" + objectName)
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
	}
	s3Writer := s3Writer{s3Client: &httpS3Client}
	err := s3Writer.Write(testFolder, "edm_security_entity_map_test.txt")
	as.NoError(err)

	dbFile, err := os.Open(dbFolder + "/" + "edm_security_entity_map_test" + "_" + time.Now().Format("2006-01-02") + ".txt")
	as.NoError(err)
	dbFile.Close()
	defer as.NoError(os.RemoveAll(dbFolder))

}