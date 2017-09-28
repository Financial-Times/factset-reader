package main

import (
	"fmt"

	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/service-status-go/gtg"
	log "github.com/Sirupsen/logrus"
)

func (h *httpHandler) factsetHealthcheck() v1a.Check {
	return v1a.Check{
		BusinessImpact:   "Unable to download the latest dataset from Factset",
		Name:             "Check connectivity to Factset",
		PanicGuide:       "TODO",
		Severity:         1,
		TechnicalSummary: "Cannot connect to Factset to be able to supply financial instruments",
		Checker:          h.checkConnectivityToFactset,
	}
}

func (h *httpHandler) amazonS3Healthcheck() v1a.Check {
	return v1a.Check{
		BusinessImpact:   "Unable to write the latest dataset to S3",
		Name:             "Check connectivity to Amazon S3",
		PanicGuide:       "TODO",
		Severity:         1,
		TechnicalSummary: "Cannot connect to Amazon S3 bucket to write the latest factset dataset",
		Checker:          h.checkConnectivityToS3,
	}
}

func (h *httpHandler) checkConnectivityToFactset() (string, error) {
	err := h.s.checkConnectivityToFactset()
	if err != nil {
		return fmt.Sprintf("Healthcheck: Unable to connect to Factset server: %v", err.Error()), err
	}
	return "", nil
}

func (h *httpHandler) checkConnectivityToS3() (string, error) {
	err := h.s.checkConnectivityToAmazonS3()
	if err != nil {
		return fmt.Sprintf("Healthcheck: Unable to connect to Amazon S3: %v", err.Error()), err
	}
	return "", nil
}

func (h *httpHandler) goodToGo() gtg.Status {
	if msg, err := h.checkConnectivityToFactset(); err != nil {
		log.Error(err)
		return gtg.Status{GoodToGo: false, Message: msg}
	}
	if msg, err := h.checkConnectivityToS3(); err != nil {
		log.Error(err)
		return gtg.Status{GoodToGo: false, Message: msg}
	}
	return gtg.Status{GoodToGo: true}
}
