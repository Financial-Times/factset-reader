package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"net/http"
	"strconv"
	"strings"
	"github.com/robfig/cron"
)

const resSeparator = ","

type httpHandler struct {
	s service
}

func main() {

	app := cli.App("Factset reader", "Reads data from factset ftp server and stores it to amazon s3")

	awsAccessKey := app.String(cli.StringOpt{
		Name:   "aws-access-key-id",
		Desc:   "s3 access key",
		EnvVar: "AWS_ACCESS_KEY_ID",
	})
	awsSecretKey := app.String(cli.StringOpt{
		Name:   "aws-secret-access-key",
		Desc:   "s3 secret key",
		EnvVar: "AWS_SECRET_ACCESS_KEY",
	})
	bucketName := app.String(cli.StringOpt{
		Name:   "bucket-name",
		Desc:   "bucket name of factset data",
		EnvVar: "BUCKET_NAME",
	})
	s3Domain := app.String(cli.StringOpt{
		Name:   "s3-domain",
		Value:  "s3.amazonaws.com",
		Desc:   "s3 domain of factset bucket",
		EnvVar: "S3_DOMAIN",
	})
	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "application port",
		EnvVar: "PORT",
	})
	factsetUser := app.String(cli.StringOpt{
		Name:   "factsetUsername",
		Desc:   "Factset username",
		EnvVar: "FACTSET_USER",
	})
	factsetKey := app.String(cli.StringOpt{
		Name:   "factsetKey",
		Desc:   "Key to ssh key",
		EnvVar: "FACTSET_KEY",
	})
	factsetFTP := app.String(cli.StringOpt{
		Name:   "factsetFTP",
		Value:  "fts-sftp.factset.com",
		Desc:   "factset ftp server address",
		EnvVar: "FACTSET_FTP",
	})
	factsetPort := app.Int(cli.IntOpt{
		Name:   "factsetPort",
		Value:  6671,
		Desc:   "Factset connection port",
		EnvVar: "FACTSET_PORT",
	})

	resources := app.String(cli.StringOpt{
		Name:   "factsetResources",
		Value: "/datafeeds/edm/edm_premium/edm_premium_full:edm_security_entity_map.txt",
		Desc:   "factset resources to be loaded",
		EnvVar: "FACTSET_RESOURCES",
	})

	app.Action = func() {
		s3 := s3Config{
			accKey:    *awsAccessKey,
			secretKey: *awsSecretKey,
			bucket:    *bucketName,
			domain:    *s3Domain,
		}

		fc := sftpConfig{
			address:  *factsetFTP,
			username: *factsetUser,
			keyPath:  *factsetKey,
			port:     *factsetPort,
		}

		fsClient := sftpClient{config: fc}
		reader := factsetReader{client: &fsClient}
		s3Client := httpS3Client{config:s3}
		writer := s3Writer{s3Client: &s3Client}

		s := service{
			reader: &reader,
			writer: &writer,
		}

		factsetRes := getResourceList(resources)
		c := cron.New()
		//run the upload every monday at 10:00 AM
		c.AddFunc("0 0 10 30 * 5", func() {
			err := s.UploadFromFactset(factsetRes)
			if err != nil {
				log.Error(err)
			}
		})
		c.Start()

		httpHandler := &httpHandler{s: s}
		listen(httpHandler, *port)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("[%v]", err)
	}
}

func getResourceList(resources string) []factsetResource {
	factsetRes := []factsetResource{}
	resList := strings.Split(resources, resSeparator)
	for _, fulRes := range resList {
		resPath := strings.Split(fulRes, ":")
		if len(resPath) == 2 {
			fr := factsetResource{
				archive:  resPath[0],
				fileName: resPath[1],
			}
			factsetRes = append(factsetRes, fr)
		}
	}
	return factsetRes
}

func listen(h *httpHandler, port int) {
	log.Infof("Listening on port: %d", port)
	r := mux.NewRouter()
	r.HandleFunc("/__health", h.health()).Methods("GET")
	r.HandleFunc("/__gtg", h.gtg()).Methods("GET")
	r.HandleFunc("/jobs", h.createJob).Methods("POST")
	err := http.ListenAndServe(":" + strconv.Itoa(port), r)
	if err != nil {
		log.Error(err)
	}
}

func (h httpHandler) createJob(w http.ResponseWriter, r *http.Request) {
	factsetRes := factsetResource{
		archive:  "/datafeeds/edm/edm_premium/edm_premium_full",
		fileName: "edm_security_entity_map.txt",
	}
	go func() {
		err := h.s.UploadFromFactset([]factsetResource{factsetRes})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
}
