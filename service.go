package main

import (
	"net/http"
	"os"
	"path"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
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
	if len([]factsetResource{}) == 0 {
		log.Warnf("Resource list not set, skipping run")
		return
	}
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

	results, err := rd.Read(res, dataFolder)

	if err != nil {
		return err
	}
	if len(results) == 0 {
		return errors.New("No results found")
	}

	for _, result := range results {
		for _, factsetFile := range result.filesToWrite {
			wr, err := NewWriter(s.wrConfig)
			if err != nil {
				return err
			}
			err = wr.Write(dataFolder, factsetFile, result.archive)
			if err != nil {
				return err
			}
		}
		defer func() {
			os.Remove(path.Join(dataFolder, result.archive))
		}()
	}

	for _, result := range results {
		for _, factsetFile := range result.filesToWrite {
			os.Remove(path.Join(dataFolder, factsetFile))

		}
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
