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
	"time"
)

type Reader interface {
	Read(fRes factsetResource, dest string) ([]string, string, error)
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

func (sfr *FactsetReader) Read(fRes factsetResource, dest string) ([]string, string, error) {
	dir, res := path.Split(fRes.archive)
	fmt.Printf("Directory is %s\n", dir)
	fmt.Printf("Res is %s\n", res)
	files, err := sfr.client.ReadDir(dir)
	fmt.Printf("Files is %s\n", files)
	if err != nil {
		return []string{}, "", err
	}

	mostRecentZipFiles, version, err := sfr.GetMostRecentZips(files, res)
	if err != nil {
		return mostRecentZipFiles, version, err
	}

	unzippedArchive := []string{}

	for _, archive := range mostRecentZipFiles {
		err = sfr.download(dir, archive, dest)
		if err != nil {
			return []string{}, version, err
		}
		factsetFiles := strings.Split(fRes.fileNames, ";")
		for _, factsetFile := range factsetFiles {
			justFileName := strings.TrimSuffix(factsetFile, ".txt")
			err = sfr.unzip(archive, justFileName, dest)
			if err != nil {
				return []string{}, version, err
			}
			unzippedArchive = append(unzippedArchive, archive)
		}

	}

	return unzippedArchive, version, err
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

func (sfr *FactsetReader) GetMostRecentZips(files []os.FileInfo, searchedFileName string) ([]string, string, error) {
	foundFile := &struct {
		minorVersion int
	}{}

	for _, file := range files {
		fmt.Printf("File Name is %s\n", file.Name())
		minorVersion, err := sfr.getMinorVersion(file.Name())
		//majorVersion, err := sfr.getMajorVersion(file.Name())
		if err!= nil {
			return []string{}, "", err
		}
		if (minorVersion > foundFile.minorVersion) {
			foundFile.minorVersion = minorVersion
		}
	}

	//foundFile.minorVersion = 1220
	fmt.Printf("Most recent version is %s\n", foundFile.minorVersion)

	fmt.Printf("SearchedFileName is %s\n", searchedFileName)
	var mostRecentZipFiles []string
	//var minorVersion = strconv.Itoa(foundFile.minorVersion)
	var minorVersion = "906"
	for _, file := range files {
		name := file.Name()
		if !strings.Contains(name, searchedFileName) {
			fmt.Printf("File name %s does not match searched file: %s\n", name, searchedFileName)
			continue
		}
		if strings.Contains(name, strconv.Itoa(foundFile.minorVersion)) {
			fmt.Printf("File names match and version %s is the same as this file %s\n", minorVersion, name)
			mostRecentZipFiles = append(mostRecentZipFiles, name)
		}
		continue
	}
	if len(mostRecentZipFiles) > 0 {
		return mostRecentZipFiles, minorVersion, nil
	}
	return mostRecentZipFiles, minorVersion, errors.New("Found no matching files with name" + searchedFileName + " and version " + minorVersion)
}

func (sfr *FactsetReader) unzip(archive string, name string, dest string) error {
	r, err := zip.OpenReader(path.Join(dest, archive))
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if !strings.Contains(f.Name, name) {
			continue
		}
		//TODO
		//FAILING HERE
		fmt.Print("Failing here?\n")
		rc, err := f.Open()
		fmt.Print("Or not\n")
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
		//if strings.Contains(archive, "full") {
		//	fullFileName := f.Name
		//	extension := filepath.Ext(fullFileName)
		//	nameWithoutExt := strings.TrimSuffix(fullFileName, extension)
		//	os.Rename(file.Name(), nameWithoutExt + "_full" + extension)
		//}
		fmt.Printf("Archive is %s\n", archive)
		fmt.Printf("FinalFileName is %s\n", file.Name())
		file.Close()
		rc.Close()

	}
	return nil
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
	fmt.Printf("Just file name is %s\n", justFileName)
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
