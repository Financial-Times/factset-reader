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
	"github.com/golang/go/src/pkg/fmt"
)

type Reader interface {
	Read(fRes factsetResource, dest string) ([]string, error)
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

func (sfr *FactsetReader) Read(fRes factsetResource, dest string) ([]string, error) {
	dir, res := path.Split(fRes.archive)
	fmt.Printf("Directory is %s\n", dir)
	fmt.Printf("Res is %s\n", res)
	files, err := sfr.client.ReadDir(dir)
	fmt.Printf("Files is %s\n", files)
	if err != nil {
		return []string{}, err
	}

	mostRecentZipFiles, err := sfr.getMostRecentZips(files, res)
	if err != nil {
		return mostRecentZipFiles, err
	}

	unzippedArchive := []string{}

	for _, archive := range mostRecentZipFiles {
		err = sfr.download(dir, archive, dest)
		if err != nil {
			return []string{}, err
		}
		factsetFiles := strings.Split(fRes.fileNames, ";")
		for _, factsetFile := range factsetFiles {
			err = sfr.unzip(archive, factsetFile, dest)
			if err != nil {
				return []string{}, err
			}
			unzippedArchive = append(unzippedArchive, archive)
		}

	}

	return unzippedArchive, err
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

func (sfr *FactsetReader) getMostRecentZips(files []os.FileInfo, searchedFileName string) ([]string, error) {
	//TODO add errors
	foundFile := &struct {
		minorVersion int
	}{}

	for _, file := range files {
		fmt.Printf("File Name is %s\n", file.Name())
		minorVersion, err := sfr.getMinorVersion(file.Name())
		if err!= nil {
			return []string{}, err
		}
		if (minorVersion > foundFile.minorVersion) {
			foundFile.minorVersion = minorVersion
		}
	}
	fmt.Printf("Most recent version is %s\n", foundFile.minorVersion)

	fmt.Printf("SearchedFileName is %s\n", searchedFileName)
	var mostRecentZipFiles []string
	var minorVersion = strconv.Itoa(foundFile.minorVersion)
	for _, file := range files {
		name := file.Name()
		if !strings.Contains(name, searchedFileName) {
			fmt.Printf("File name %s does not match searched file: %s\n", name, searchedFileName)
			continue
		}
		if strings.Contains(name, strconv.Itoa(foundFile.minorVersion)) {
			fmt.Printf("File names match and version %s is different to this file %s\n", minorVersion, name)
			mostRecentZipFiles = append(mostRecentZipFiles, name)
		}
		continue
	}
	if len(mostRecentZipFiles) > 0 {
		return mostRecentZipFiles, nil
	}
	return mostRecentZipFiles, errors.New("Found no matching files with name" + searchedFileName + " and version " + minorVersion)
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
