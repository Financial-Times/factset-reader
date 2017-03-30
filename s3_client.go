package main

import (
	"bytes"
	"github.com/minio/minio-go"
)

type S3Client interface {
	PutObject(objectName string, filePath string) (int64, error)
	PutData(objectName string, data []byte) error
	BucketExists(bucket string) (bool, error)
}

type HTTPS3Client struct {
	client *minio.Client
	bucket string
}

func NewS3Client(config s3Config) (S3Client, error) {
	mClient, err := minio.New(config.domain, config.accKey, config.secretKey, true)
	return &HTTPS3Client{client: mClient, bucket: config.bucket}, err
}

func (s3 *HTTPS3Client) PutObject(objectName string, filePath string) (int64, error) {
	size, err := s3.client.FPutObject(s3.bucket, objectName, filePath, "application/octet-stream")
	return size, err
}

func (s3 *HTTPS3Client) PutData(objectName string, data []byte) (err error) {
	_, err = s3.client.PutObject(s3.bucket, objectName, bytes.NewReader(data), "text/plain")
	return err
}

func (s3 *HTTPS3Client) BucketExists(bucket string) (bool, error) {
	return s3.client.BucketExists(bucket)
}
