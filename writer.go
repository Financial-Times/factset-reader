package main

import (
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Writer interface {
	Write(src string, resName string) error
}

type S3Writer struct {
	s3Client S3Client
}

func NewWriter(config s3Config) (Writer, error) {
	s3, err := NewS3Client(config)
	return &S3Writer{s3Client: s3}, err
}

func (s3w *S3Writer) Write(src string, resName string) error {
	name := s3w.gets3ResName(resName)
	p := path.Join(src, resName)
	n, err := s3w.s3Client.PutObject(name, p)
	if err != nil {
		return err
	}
	log.Infof("Uploaded file [%s] of size [%d] successfully", name, n)
	return nil
}

func (s3w *S3Writer) gets3ResName(res string) string {
	fileData := strings.Split(res, ".")
	date := time.Now().Format("2006-01-02")
	if len(fileData) >= 2 {
		name := fileData[0]
		ext := fileData[1]
		return name + "_" + date + "." + ext
	} else if len(fileData) == 1 {
		name := fileData[0]
		return name + "_" + date
	}
	return res
}
