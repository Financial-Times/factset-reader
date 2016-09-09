package main

import (
	log "github.com/Sirupsen/logrus"
	"time"
	"os"
)

type service struct {
	reader reader
	writer writer
}

func (s service) UploadFromFactset(res []factsetResource) error {
	defer os.RemoveAll(dataFolder)

	err := s.reader.Init()
	if err != nil {
		return err
	}
	defer s.reader.Close()

	for _, r := range res {
		start := time.Now()
		log.Infof("Loading resource [%s]", r)
		err := s.reader.ReadRes(r, dataFolder)
		if err != nil {
			log.Errorf("Error while reading resource [%s]", r)
			return err
		}
		log.Infof("Resource [%s] was succesfully read from Factset in %d", r.fileName, time.Since(start))
		err = s.writer.Write(r.fileName, dataFolder)
		if err != nil {
			log.Errorf("Error while writing resource [%s]", r)
			return err
		}
		log.Infof("Finished writting resource [%s] to s3 in %d", r, time.Since(start))
	}
	return nil
}
