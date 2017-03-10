package main

import (
	"net/http"
	"os"
	"path"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/golang/go/src/pkg/fmt"
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
	wg.Wait()
}

func (s service) fetchResource(res factsetResource) error {

	rd, err := NewReader(s.rdConfig)
	if err != nil {
		return err
	}
	defer rd.Close()

	log.Infof("Loading resource [%s]", res)

	results, err := rd.Read(res, dataFolder)

	if err != nil {
		return err
	}
	if len(results) == 0 {
		return errors.New("No results found")
	}

	for _, result := range results {
		for _, factsetFile := range result.filesToWrite {

			log.Infof("Resource [%s] was succesfully read from Factset", factsetFile)

			wr, err := NewWriter(s.wrConfig)
			if err != nil {
				return err
			}
			err = wr.Write(dataFolder, factsetFile, result.archive)
			if err != nil {
				return err
			}
			defer func() {
				fmt.Printf("Deleting %s from local directory\n", factsetFile)
				os.Remove(path.Join(dataFolder, factsetFile))
			}()
		}
		defer func() {
			fmt.Printf("Deleting %s from local directory\n", result.archive)
			os.Remove(path.Join(dataFolder, result.archive))
		}()
	}


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
