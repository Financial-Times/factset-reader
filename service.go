package main

import (
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"path/filepath"
	"strings"
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
	start := time.Now()

	rd, err := NewReader(s.rdConfig)
	if err != nil {
		return err
	}
	defer rd.Close()

	log.Infof("Loading resource [%s]", res)
	fileName, err := rd.Read(res, dataFolder)
	if err != nil {
		return err
	}

	fullVersion, err := rd.GetFullVersion(fileName)
	if err != nil {
		return err
	}

	factsetFiles := strings.Split(res.fileNames, ";")

	for _, factsetFile := range factsetFiles {
		extension := filepath.Ext(factsetFile)
		nameWithoutExt := strings.TrimSuffix(factsetFile, extension)
		fileNameOnS3 := nameWithoutExt + "_" + fullVersion + extension

		log.Infof("Resource [%s] was succesfully read from Factset in %d", fileName, time.Since(start).String())

		wr, err := NewWriter(s.wrConfig)
		if err != nil {
			return err
		}
		err = wr.Write(dataFolder, factsetFile, fileNameOnS3)
		if err != nil {
			return err
		}
		defer func() {
			os.Remove(path.Join(dataFolder, fileNameOnS3))
		}()

	}

	defer func() {
		os.Remove(path.Join(dataFolder, fileName))
	}()
	log.Infof("Finished writting resource [%s] to S3 in %d", res, time.Since(start).String())
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
