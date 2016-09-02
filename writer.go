package main

import (
	log "github.com/Sirupsen/logrus"
	"strings"
	"time"
	"os"
	"io"
	"fmt"
	"github.com/rlmcpherson/s3gof3r"
)

type writer interface {
	Write(resName string) error
}

type s3Writer struct {
	config s3Config
}

//func (s3w s3Writer) Write(resName string) error {
//	s3Client, err := minio.New(s3w.config.domain, s3w.config.accKey, s3w.config.secretKey, true)
//	if err != nil {
//		return err
//	}
//
//	name := gets3ResName(resName)
//	//n, err := s3Client.FPutObject(s3w.config.bucket, name, dataFolder + "/" + resName, "")
//	n, err := s3Client.GetBucketLocation(s3w.config.bucket)
//	if err != nil {
//		log.Error(err)
//		return err
//	}
//	log.Infof("Uploaded file [%s] of size [%s] successfully", name, n)
//	return nil
//}

func (s3w s3Writer) Write(resName string) error {
	k, err := s3gof3r.EnvKeys() // get S3 keys from environment
	if err != nil {
		return err
	}
	// Open bucket to put file into
	s3 := s3gof3r.New(s3w.config.domain, k)
	b := s3.Bucket(s3w.config.bucket)

	// open file to upload
	file, err := os.Open(dataFolder + "/" + resName)
	if err != nil {
		return err
	}

	// Open a PutWriter for upload
	n := gets3ResName(resName)
	w, err := b.PutWriter(n, nil, nil)
	if err != nil {
		log.Error(err)
		return err
	}
	if _, err = io.Copy(w, file); err != nil {
		// Copy into S3
		log.Error(err)
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	fmt.Println(n)
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
