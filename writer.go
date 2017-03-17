package main

import (
	"path"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Writer interface {
	Write(src string, fileName string) error
}

type S3Writer struct {
	s3Client S3Client
}

func NewWriter(config s3Config) (Writer, error) {
	s3, err := NewS3Client(config)
	return &S3Writer{s3Client: s3}, err
}

func (s3w *S3Writer) Write(src string, fileName string) error {
	log.Infof("Writing file [%s]", fileName)
	s3ResFilePath := time.Now().Format("2006-01-02") + "/" + fileName
	p := path.Join(src, fileName)
	n, err := s3w.s3Client.PutObject(s3ResFilePath, p)
	if err != nil {
		return err
	}
	log.Infof("Uploaded file [%s] of size [%d] successfully", s3ResFilePath, n)
	return nil
}
