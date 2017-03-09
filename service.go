package main

import (
	"net/http"
	"os"
	"path"
	"sync"

	log "github.com/Sirupsen/logrus"
	"path/filepath"
	"strings"
	"fmt"
)

type service struct {
	rdConfig sftpConfig
	wrConfig s3Config
	files    []factsetResource
}

func (s service) forceImport(rw http.ResponseWriter, req *http.Request) {
	go s.Fetch()
	log.Info("Triggered fetching")
}

func (s service) Fetch() {
	res := s.files

	errorsCh := make(chan error)
	var wg sync.WaitGroup
	wg.Add(len(res))

	for _, r := range res {
		go func(res factsetResource) {
			defer wg.Done()
			err := s.fetchResource(res)
			errorsCh <- err
		}(r)
	}

	go handleErrors(errorsCh)
	log.Info("Finished writing files to s3")
	wg.Wait()
}

func (s service) fetchResource(res factsetResource) error {

	rd, err := NewReader(s.rdConfig)
	if err != nil {
		return err
	}
	defer rd.Close()

	log.Infof("Loading resource [%s]", res)
	archive, filesToWrite, version, err := rd.Read(res, dataFolder)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	//factsetFiles := strings.Split(res.fileNames, ";")
	fmt.Printf("Unzipped archive is %s\n", archive)
	//justFolder := strings.TrimSuffix(archive, ".zip")
	for _, factsetFile := range filesToWrite {
		extension := filepath.Ext(factsetFile)
		nameWithoutExt := strings.TrimSuffix(factsetFile, extension)
		fileNameOnS3 := nameWithoutExt + "_" + version + extension
		fmt.Printf("FactsetFile is %s\n", factsetFile)
		fmt.Printf("Extension is %s\n", extension)
		fmt.Printf("NameWithoutExt is %s\n", nameWithoutExt)
		fmt.Printf("FileNameOnS3 is %s\n", fileNameOnS3)

		log.Infof("Resource [%s] was succesfully read from Factset", factsetFile)

		wr, err := NewWriter(s.wrConfig)
		if err != nil {
			return err
		}
		err = wr.Write(dataFolder, factsetFile, fileNameOnS3, archive)
		if err != nil {
			return err
		}
		defer func() {
			os.Remove(path.Join(dataFolder, fileNameOnS3))
		}()
	}
	defer func() {
		os.Remove(path.Join(dataFolder, archive))
	}()


	log.Infof("Finished writing resource [%s] to S3", res)
	return nil
}

func handleErrors(errors chan error) {
	for e := range errors {
		if e != nil {
			log.Error(e)
		}
	}
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
