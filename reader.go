package main

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"errors"
	log "github.com/Sirupsen/logrus"
	"strconv"
)

type Reader interface {
	Read(fRes factsetResource, dest string) (string, error)
	GetFullVersion(filename string) (string, error)
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
	factsetFiles := strings.Split(fRes.fileNames, ";")
	for _, factsetFile := range factsetFiles {
		err = sfr.unzip(lastVers, factsetFile, dest)
		if err != nil {
			return lastVers, err
		}
	}

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

func (sfr *FactsetReader) getLastVersion(files []os.FileInfo, searchedFileName string) (string, error) {
	foundFile := &struct {
		name         string
		majorVersion int
		minorVersion int
	}{}

	for _, file := range files {
		name := file.Name()
		if !strings.Contains(name, searchedFileName) {
			continue
		}

		fullVersion, err := sfr.GetFullVersion(name)
		if err != nil {
			continue
		}

		majorVersion, _ := sfr.getMajorVersion(fullVersion)
		minorVersion, _ := sfr.getMinorVersion(fullVersion)

		if (majorVersion > foundFile.majorVersion) ||
			(majorVersion == foundFile.majorVersion && minorVersion > foundFile.minorVersion) {
			foundFile.name = name
			foundFile.majorVersion = majorVersion
			foundFile.minorVersion = minorVersion
		}
	}

	if len(foundFile.name) == 0 {
		return foundFile.name, errors.New("Could not find any last version file matching filename: " + searchedFileName)
	}

	return foundFile.name, nil
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

func (sfr *FactsetReader) GetFullVersion(filename string) (string, error) {
	regex := regexp.MustCompile("v[0-9]+_full_[0-9]+\\.zip|txt$")

	foundMatches := regex.FindStringSubmatch(filename)
	if foundMatches == nil {
		return "", errors.New("The full version is missing or not correctly specified! Filename was " + filename)
	}
	if len(foundMatches) > 1 {
		return "", errors.New("More than 1 full version found!")
	}

	fullVersion := strings.TrimSuffix(foundMatches[0], ".zip")
	return fullVersion, nil
}

func (sfr *FactsetReader) getMajorVersion(fullVersion string) (int, error) {
	regex := regexp.MustCompile("^v[0-9]+")
	foundMatches := regex.FindStringSubmatch(fullVersion)
	if foundMatches == nil {
		return -1, errors.New("The major version is missing or not correctly specified!")
	}
	if len(foundMatches) > 1 {
		return -1, errors.New("More than 1 major version found!")
	}
	majorVersion, _ := strconv.Atoi(strings.TrimPrefix(foundMatches[0], "v"))
	return majorVersion, nil
}

func (sfr *FactsetReader) getMinorVersion(fullVersion string) (int, error) {
	regex := regexp.MustCompile("_[0-9]+$")
	foundMatches := regex.FindStringSubmatch(fullVersion)
	if foundMatches == nil {
		return -1, errors.New("The minor version is missing or not correctly specified!")
	}
	if len(foundMatches) > 1 {
		return -1, errors.New("More than 1 minor version found!")
	}
	minorVersion, _ := strconv.Atoi(strings.TrimPrefix(foundMatches[0], "_"))
	return minorVersion, nil
}
