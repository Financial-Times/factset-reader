package main

import (
	"github.com/minio/minio-go"
)

type s3Client interface {
	PutObject(objectName string, filePath string) (int64, error)
}

type httpS3Client struct {
	config s3Config
}

func (client *httpS3Client) PutObject(objectName string, filePath string) (int64, error) {
	c := client.config
	s3Client, err := minio.New(c.domain, c.accKey, c.secretKey, true)
	if err != nil {
		return 0, err
	}
	size, err := s3Client.FPutObject(c.bucket, objectName, filePath, "application/octet-stream")
	return size, err
}
