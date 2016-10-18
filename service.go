package main

import (
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
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

	errors := make(chan error)
	var wg sync.WaitGroup
	wg.Add(len(res))

	for _, r := range res {
		go func(res factsetResource) {
			defer wg.Done()
			start := time.Now()

			rd, err := NewReader(s.rdConfig)
			if err != nil {
				errors <- err
				return
			}
			defer rd.Close()

			log.Infof("Loading resource [%s]", res)
			fileName, err := rd.Read(res, dataFolder)
			if err != nil {
				errors <- err
				return
			}

			defer func() {
				os.Remove(path.Join(dataFolder, fileName))
				os.Remove(path.Join(dataFolder, res.fileName))
			}()

			log.Infof("Resource [%s] was succesfully read from Factset in %d", res.fileName, time.Since(start))

			wr, err := NewWriter(s.wrConfig)
			if err != nil {
				errors <- err
				return
			}
			err = wr.Write(dataFolder, res.fileName)
			if err != nil {
				errors <- err
				return
			}
			log.Infof("Finished writting resource [%s] to S3 in %d", res, time.Since(start))
			errors <- nil
		}(r)
	}

	go func() {
		for e := range errors {
			if e != nil {
				log.Error(e)
			}
		}
	}()

	wg.Wait()
}

func (s service) checkConnectivityToFactset() error {
	reader, err := NewReader(s.rdConfig)
	reader.Close()
	return err
}

func (s service) checkConnectivityToAmazonS3() error {
	_, err := NewWriter(s.wrConfig)
	return err
}
