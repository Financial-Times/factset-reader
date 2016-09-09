package main

import (
	log "github.com/Sirupsen/logrus"
	"path"
	"strings"
	"time"
)

type writer interface {
	Write(src string, resName string) error
}

type s3Writer struct {
	s3Client s3Client
}

func (s3w *s3Writer) Write(src string, resName string) error {
	name := s3w.gets3ResName(resName)
	p := path.Join(src, resName)
	n, err := s3w.s3Client.PutObject(name, p)
	if err != nil {
		return err
	}
	log.Infof("Uploaded file [%s] of size [%d] successfully", name, n)
	return nil
}

func (s3w *s3Writer) gets3ResName(res string) string {
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
