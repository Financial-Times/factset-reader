package main

import (
	"archive/zip"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type reader interface {
	Read(fRes factsetResource, dest string) (string, error)
	Close()
}

type factsetReader struct {
	client factsetClient
}

func NewReader(config sftpConfig) (reader, error) {
	fc := &sftpClient{config: config}
	err := fc.Init()
	return &factsetReader{client: fc}, err
}

func (sfr *factsetReader) Close() {
	sfr.client.Close()
}

func (sfr *factsetReader) Read(fRes factsetResource, dest string) (string, error) {
	dir, res := path.Split(fRes.archive)
	files, err := sfr.client.ReadDir(dir)
	if err != nil {
		return "", err
	}

	lastVers, err := sfr.getLastVersion(files, res)
	if err != nil {
		return lastVers, err
	}

	err = sfr.download(dir, lastVers, dest)
	if err != nil {
		return lastVers, err
	}

	err = sfr.unzip(lastVers, fRes.fileName, dest)
	return lastVers, err
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
	recFile := &struct {
		name string
		vers int
	}{}

	r := regexp.MustCompile("[0-9]+")
	for _, file := range files {
		name := file.Name()
		if strings.Contains(name, searchedRes) {
			v, err := strconv.Atoi(r.FindStringSubmatch(name)[0])
			if err != nil {
				return "", err
			}
			if recFile == nil {
				recFile.name = name
				recFile.vers = v
			} else {
				if v > recFile.vers {
					recFile.name = name
					recFile.vers = v
				}

			}
		}
	}
	return recFile.name, nil
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
