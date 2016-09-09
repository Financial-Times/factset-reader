package main

import (
	"archive/zip"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"strings"
	"time"
	"path"
)

type reader interface {
	Init() error
	Close()
	ReadRes(fRes factsetResource, dest string) error
}

type factsetReader struct {
	client factsetClient
}

func (sfr *factsetReader) Init() error {
	return sfr.client.Init()
}

func (sfr *factsetReader) Close() {
	sfr.client.Close()
}

func (sfr *factsetReader) ReadRes(fRes factsetResource, dest string) error {
	dir, res := path.Split(fRes.archive)
	files, err := sfr.client.ReadDir(dir)
	if err != nil {
		return err
	}

	lastVers, err := sfr.getLastVersion(files, res)
	if err != nil {
		return err
	}

	err = sfr.download(dir, lastVers, dest)
	if err != nil {
		return err
	}

	err = sfr.unzip(lastVers, fRes.fileName, dest)
	return err
}

func (sfr *factsetReader) download(filePath string, fileName string, dest string) error {
	fullName := path.Join(filePath, fileName)
	log.Infof("Downloading file [%s]", fullName)

	err := sfr.client.Download(fullName, dest)
	if err != nil {
		return err
	}

	log.Infof("File [%s] was downloaded successfully", fullName)
	return nil
}

func (sfr *factsetReader) getLastVersion(files []os.FileInfo, searchedRes string) (string, error) {
	lastFile := &struct {
		name         string
		lastModified time.Time
	}{}

	for _, file := range files {
		name := file.Name()
		if strings.Contains(name, searchedRes) {
			if lastFile == nil {
				lastFile.name = name
				lastFile.lastModified = file.ModTime()
			} else {
				if file.ModTime().After(lastFile.lastModified) {
					lastFile.name = name
					lastFile.lastModified = file.ModTime()
				}
			}
		}
	}
	return lastFile.name, nil
}

func (sfr *factsetReader) unzip(archive string, name string, dest string) error {
	r, err := zip.OpenReader(path.Join(dest, archive))
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if name == f.Name {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			file, err := os.Create(path.Join(dest, f.Name))
			if err != nil {
				return err
			}
			_, err = io.Copy(file, rc)
			if err != nil {
				return err
			}
			file.Close()
			rc.Close()
		}
	}
	return nil
}
