package main

import (
	"net/http"
	"os"
	"path"

	"archive/zip"
	"errors"
	log "github.com/Sirupsen/logrus"
	"io"
	"path/filepath"
	"strings"
)

type service struct {
	rdConfig sftpConfig
	wrConfig s3Config
	files    []factsetResource
	weekly   bool
}

func (s service) forceImportWeekly(rw http.ResponseWriter, req *http.Request) {
	s.weekly = true
	go s.fetchResources(s.files)
	log.Info("Triggered fetching last weekly files")
}

func (s service) forceImport(rw http.ResponseWriter, req *http.Request) {
	go s.fetchResources(s.files)
	log.Info("Triggered fetching most recently released files")
}

func (s service) fetchResources(resources []factsetResource) error {
	rd, err := NewReader(s.rdConfig)
	if err != nil {
		return err
	}
	defer rd.Close()

	if _, err := os.Stat(dataFolder + "/" + weekly); os.IsNotExist(err) {
		os.Mkdir(dataFolder+"/"+weekly, 0755)
	}

	if _, err := os.Stat(dataFolder + "/" + daily); os.IsNotExist(err) {
		os.Mkdir(dataFolder+"/"+daily, 0755)
	}

	var fileCollection []zipCollection
	for _, res := range resources {
		requestedFiles, _ := rd.Read(res, dataFolder, s.weekly)
		for _, requestedFile := range requestedFiles {
			fileCollection = append(fileCollection, requestedFile)
		}
	}

	filesToWrite, err := s.sortAndZipFiles(fileCollection)

	if err != nil {
		return err
	}
	if len(fileCollection) == 0 {
		return errors.New("Did not find any matching files")
	}

	wr, err := NewWriter(s.wrConfig)
	if err != nil {
		return err
	}
	for _, fileToWrite := range filesToWrite {
		err = wr.Write(dataFolder, fileToWrite)
		if err != nil {
			return err
		}
	}

	defer s.cleanUpWorkingDirectory(fileCollection, filesToWrite)

	return nil
}

func zipFilesForUpload(fileTypes string) (string, error) {
	var workingDir string
	if fileTypes == weekly {
		workingDir = dataFolder + "/" + weekly
	} else if fileTypes == daily {
		workingDir = dataFolder + "/" + daily
	} else {
		return "", errors.New("Invalid local directory provided")
	}

	zipFileName := fileTypes + ".zip"
	zipFile, err := os.Create(dataFolder + "/" + zipFileName)
	defer zipFile.Close()

	if err != nil {
		return zipFileName, err
	}
	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	info, err := os.Stat(workingDir)
	if err != nil {
		return zipFileName, err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(workingDir)
	}

	filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, workingDir))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		if file.Name() != fileTypes {
			_, err = io.Copy(writer, file)
		}
		return err

	})

	if err != nil {
		return zipFileName, err
	}
	log.Infof("Created zip of todays %s files", fileTypes)
	return zipFileName, nil
}

func (s service) sortAndZipFiles(colls []zipCollection) ([]string, error) {
	var weeklyFiles []string
	var dailyFiles []string
	var filesToWrite []string
	for _, coll := range colls {
		for _, file := range coll.filesToWrite {
			if strings.Contains(file, "update") || strings.Contains(file, "delete") {
				dailyFiles = append(dailyFiles, file)
			} else {
				weeklyFiles = append(weeklyFiles, file)
			}
		}
	}

	if len(dailyFiles) == 0 && len(weeklyFiles) == 0 {
		return filesToWrite, errors.New("There are no files to write")
	}

	if s.weekly == true {
		weeklyFileName, err := zipFilesForUpload(weekly)
		if err != nil {
			return filesToWrite, err
		}
		filesToWrite = append(filesToWrite, weeklyFileName)
		return filesToWrite, err
	} else if len(weeklyFiles) == 0 {
		dailyFileName, err := zipFilesForUpload(daily)
		if err != nil {
			return filesToWrite, err
		}
		filesToWrite = append(filesToWrite, dailyFileName)
		return filesToWrite, err
	} else {
		dailyFileName, err := zipFilesForUpload(daily)
		if err != nil {
			return filesToWrite, err
		}
		filesToWrite = append(filesToWrite, dailyFileName)
		weeklyFileName, err := zipFilesForUpload(weekly)
		if err != nil {
			return filesToWrite, err
		}
		filesToWrite = append(filesToWrite, weeklyFileName)
		return filesToWrite, err
	}

	return []string{}, nil
}

func (s service) cleanUpWorkingDirectory(fileCollection []zipCollection, filesToWrite []string) {
	for _, result := range fileCollection {
		for _, factsetFile := range result.filesToWrite {
			if strings.Contains(result.archive, "full") {
				os.Remove(path.Join(dataFolder+"/"+weekly, factsetFile))
			} else {
				os.Remove(path.Join(dataFolder+"/"+daily, factsetFile))
			}

		}
		os.Remove(path.Join(dataFolder, result.archive))
	}

	for _, fileToWrite := range filesToWrite {
		os.Remove(path.Join(dataFolder, fileToWrite))
	}

	os.Remove(path.Join(dataFolder + "/" + daily + "/"))
	os.Remove(path.Join(dataFolder + "/" + daily))
	os.Remove(path.Join(dataFolder + "/" + weekly + "/"))
	os.Remove(path.Join(dataFolder + "/" + weekly))
}

func (s service) checkConnectivityToFactset() error {
	reader, err := NewReader(s.rdConfig)
	if reader != nil {
		defer reader.Close()
	}
	return err
}

func (s service) checkConnectivityToAmazonS3() error {
	s3, err := NewS3Client(s.wrConfig)
	if err != nil {
		return err
	}
	_, err = s3.BucketExists(s.wrConfig.bucket)
	if err != nil {
		return err
	}
	return nil
}
