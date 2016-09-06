package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/minio/minio-go"
	"strings"
	"time"
)

type writer interface {
	Write(resName string) error
}

type s3Writer struct {
	config s3Config
}

func (s3w s3Writer) Write(resName string) error {
	s3Client, err := minio.New(s3w.config.domain, s3w.config.accKey, s3w.config.secretKey, true)
	if err != nil {
		return err
	}

	name := gets3ResName(resName)
	n, err := s3Client.FPutObject(s3w.config.bucket, name, dataFolder+"/"+resName, "")
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("Uploaded file [%s] of size [%d] successfully", name, n)
	return nil
}

func gets3ResName(res string) string {
	fileData := strings.Split(res, ".")
	date := time.Now().Format("2006-01-02")
	if len(fileData) >= 2 {
		name := fileData[0]
		ext := fileData[1]
		return name + "_" + date + "." + ext
	} else if len(fileData) == 1 {
		name := fileData[0]
		return name + date
	}
	return res
}
