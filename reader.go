package main

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Reader interface {
	Read(fRes factsetResource, dest string) (string, error)
	Close()
}

type FactsetReader struct {
	client FactsetClient
}

func NewReader(config sftpConfig) (Reader, error) {
	fc := &SFTPClient{config: config}
	err := fc.Init()
	return &FactsetReader{client: fc}, err
}

func (sfr *FactsetReader) Close() {
	if sfr.client != nil {
		sfr.client.Close()
	}
}

func (sfr *FactsetReader) Read(fRes factsetResource, dest string) (string, error) {
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

func (sfr *FactsetReader) download(filePath string, fileName string, dest string) error {
	fullName := path.Join(filePath, fileName)
	log.Infof("Downloading file [%s]", fullName)

	err := sfr.client.Download(fullName, dest)
	if err != nil {
		return err
	}

	log.Infof("File [%s] was downloaded successfully", fullName)
	return nil
}

func (sfr *FactsetReader) getLastVersion(files []os.FileInfo, searchedRes string) (string, error) {
	recFile := &struct {
		name string
		vers int
	}{}

	r := regexp.MustCompile("[0-9]+\\.zip$")
	for _, file := range files {
		name := file.Name()
		if !strings.Contains(name, searchedRes) {
			continue
		}
		s := r.FindStringSubmatch(name)[0]

		v, err := strconv.Atoi(strings.TrimSuffix(s, ".zip"))
		if err != nil {
			return "", err
		}

		if recFile.name == "" {
			recFile.name = name
			recFile.vers = v
		} else if v > recFile.vers {
			recFile.name = name
			recFile.vers = v
		}
	}
	return recFile.name, nil
}

func (sfr *FactsetReader) unzip(archive string, name string, dest string) error {
	r, err := zip.OpenReader(path.Join(dest, archive))
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if name != f.Name {
			continue
		}
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
	return nil
}
