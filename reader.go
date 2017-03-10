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
	"time"
	"github.com/golang/go/src/pkg/fmt"
)

type Reader interface {
	Read(fRes factsetResource, dest string) ([]zipCollection, error)
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

func (sfr *FactsetReader) Read(fRes factsetResource, dest string) ([]zipCollection, error) {
	results := []zipCollection{}
	dir, res := path.Split(fRes.archive)
	files, err := sfr.client.ReadDir(dir)
	if err != nil {
		return results, err
	}

	mostRecentZipFiles, err := sfr.GetMostRecentZips(files, res)
	if err != nil {
		return results, err
	}

	for _, archive := range mostRecentZipFiles {
		fmt.Printf("Archive is %s\n", archive)
		filesToWrite := []string{}
		err = sfr.download(dir, archive, dest)
		if err != nil {
			return results, err
		}
		factsetFiles := strings.Split(fRes.fileNames, ";")
		fmt.Printf("Files to write are %s\n", factsetFiles)
		filesToWrite, err = sfr.unzip(archive, factsetFiles, dest)
		if err != nil {
			return results, err
		}

		results = append(results, zipCollection{archive:archive,filesToWrite:filesToWrite})
	}

	return results, err
}

func (sfr *FactsetReader) download(filePath string, fileName string, dest string) error {
	start := time.Now()
	fullName := path.Join(filePath, fileName)
	log.Infof("Downloading file [%s]", fullName)

	err := sfr.client.Download(fullName, dest)
	if err != nil {
		return err
	}

	log.Infof("File [%s] was downloaded successfully in %s", fullName, time.Since(start).String())
	return nil
}

func (sfr *FactsetReader) GetMostRecentZips(files []os.FileInfo, searchedFileName string) ([]string, error) {
	foundFile := &struct {
		name         string
		majorVersion int
		minorVersion int
	}{}

	for _, file := range files {
		minorVersion, err := sfr.getMinorVersion(file.Name())
		if err!= nil {
			return []string{}, err
		}
		majorVersion, err := sfr.getMajorVersion(file.Name())
		if err!= nil {
			return []string{}, err
		}
		if (majorVersion > majorVersion) ||
			(majorVersion == majorVersion && minorVersion > foundFile.minorVersion) {
			foundFile.name = file.Name()
			foundFile.majorVersion = majorVersion
			foundFile.minorVersion = minorVersion
		}
	}

	var mostRecentZipFiles []string
	var minorVersion = strconv.Itoa(foundFile.minorVersion)
	var majorVersion = strconv.Itoa(foundFile.majorVersion)
	for _, file := range files {
		name := file.Name()
		if !strings.Contains(name, searchedFileName) {
			continue
		}
		if strings.Contains(name, strconv.Itoa(foundFile.minorVersion)) && strings.Contains(name, strconv.Itoa(foundFile.majorVersion)) {
			mostRecentZipFiles = append(mostRecentZipFiles, name)
		}
		continue
	}
	if len(mostRecentZipFiles) > 0 {
		return mostRecentZipFiles, nil
	}
	return mostRecentZipFiles, errors.New("Found no matching files with name: " + searchedFileName + ", major version: " + majorVersion + ", or minor version: " + minorVersion)
}

func (sfr *FactsetReader) unzip(archive string, factsetFiles []string, dest string) ([]string,error) {
	r, err := zip.OpenReader(path.Join(dest, archive))
	if err != nil {
		return []string{}, err
	}
	defer r.Close()
	filesToWrite := []string{}

	for _, f := range r.File {
		for _, factsetFile := range factsetFiles {
			justFileName := strings.TrimSuffix(factsetFile, ".txt")
			fmt.Printf("Just file name %s\n", justFileName)
			fmt.Printf("File in archive is %s\n", f.Name)
			if !strings.Contains(f.Name, justFileName) {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return []string{}, err
			}
			file, err := os.Create(path.Join(dest, f.Name))
			if err != nil {
				return []string{}, err
			}
			_, err = io.Copy(file, rc)
			if err != nil {
				return []string{}, err
			}
			file.Close()
			rc.Close()
			filesToWrite = append(filesToWrite, strings.TrimPrefix(file.Name(), "data/"))
		}
	}
	return filesToWrite, nil
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
	justFileName := strings.TrimSuffix(fullVersion, ".zip")
	foundMatches := regex.FindStringSubmatch(justFileName)
	if foundMatches == nil {
		return -1, errors.New("The minor version is missing or not correctly specified!")
	}
	if len(foundMatches) > 1 {
		return -1, errors.New("More than 1 minor version found!")
	}
	minorVersion, _ := strconv.Atoi(strings.TrimPrefix(foundMatches[0], "_"))
	return minorVersion, nil
}
