package main

import (
	"github.com/minio/minio-go"
)

type s3Client interface {
	PutObject(objectName string, filePath string) (int64, error)
}

type httpS3Client struct {
	client *minio.Client
	bucket string
}

func NewS3Client(config s3Config) (s3Client, error) {
	mClient, err := minio.New(config.domain, config.accKey, config.secretKey, true)
	return &httpS3Client{client: mClient, bucket: config.bucket}, err
}

func (s3 *httpS3Client) PutObject(objectName string, filePath string) (int64, error) {
	size, err := s3.client.FPutObject(s3.bucket, objectName, filePath, "application/octet-stream")
	return size, err
}
