package main

import (
	"os"
	"time"
)

type sftpClientMock struct {
	readDirMock  func(dir string) ([]os.FileInfo, error)
	downloadMock func(fileName string, dest string) error
	initMock     func() error
	closeMock    func()
}

type fileInfoMock struct {
	name  string
	size  int64
	mode  os.FileMode
	mtime time.Time
	sys   interface{}
}

type httpS3ClientMock struct {
	putObjectMock    func(objectName string, filePath string) (int64, error)
	bucketExistsMock func(bucket string) (bool, error)
}

func (s *sftpClientMock) ReadDir(dir string) ([]os.FileInfo, error) {
	return s.readDirMock(dir)
}

func (s *sftpClientMock) Download(fileName string, dest string) error {
	return s.downloadMock(fileName, dest)
}

func (s *sftpClientMock) Init() error {
	return s.initMock()
}

func (s *sftpClientMock) Close() {
	s.closeMock()
}

func (fi fileInfoMock) Name() string {
	return fi.name
}

func (fi fileInfoMock) Size() int64 {
	return fi.size
}

func (fi fileInfoMock) Mode() os.FileMode {
	return fi.mode
}

func (fi fileInfoMock) ModTime() time.Time {
	return fi.mtime
}

func (fi fileInfoMock) IsDir() bool {
	return fi.Mode().IsDir()
}

func (fi fileInfoMock) Sys() interface{} {
	return fi.sys
}

func (s3w *httpS3ClientMock) PutObject(objectName string, filePath string) (int64, error) {
	return s3w.putObjectMock(objectName, filePath)
}

func (s3w *httpS3ClientMock) BucketExists(bucket string) (bool, error) {
	return s3w.bucketExistsMock(bucket)
}
