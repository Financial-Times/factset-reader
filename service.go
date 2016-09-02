package main

import (
	log "github.com/Sirupsen/logrus"
	"time"
)

type service struct {
	reader reader
	writer writer
}

func (s service) UploadFromFactset(res []factsetResource) error {
	//defer os.RemoveAll(dataFolder)

	for _, r := range res {
		start := time.Now()
		log.Infof("Starting loading resource [%s]", r)
		err := s.reader.ReadRes(r)
		if err != nil {
			log.Errorf("Error while reading resource [%s]", r)
			return err
		}
		log.Infof("Resource [%s] was succesfully readed from Factset in %d", r.fileName, time.Since(start))
		err = s.writer.Write(r.fileName)
		if err != nil {
			log.Errorf("Error while writing resource [%s]", r)
			return err
		}
		log.Infof("Finished loading resource [%s] to s3 in %d", r, time.Since(start))
	}
	return nil
}
