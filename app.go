package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/robfig/cron"
	"net/http"
	"strconv"
	"strings"
)

const serSeparator = ","

type httpHandler struct {
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
	factsetUser := app.String(cli.StringOpt{
		Name:   "factsetUsername",
		Desc:   "factset username",
		EnvVar: "FACTSET_USER",
	})

	factsetPasswd := app.String(cli.StringOpt{
		Name:   "factsetPasswd",
		Desc:   "factset password",
		EnvVar: "FACTSET_PWD",
	})

	factsetFTP := app.String(cli.StringOpt{
		Name:   "factsetFTP",
		Value:  "fts-sftp.factset.com",
		Desc:   "factset ftp server address",
		EnvVar: "FACTSET_FTP",
	})

	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "application port",
		EnvVar: "PORT",
	})

	resources := app.String(cli.StringOpt{
		Name:   "factsetResources",
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

		fc := factsetConfig{
			address:  *factsetFTP,
			username: *factsetUser,
			password: *factsetPasswd,
		}

		reader := factsetReader{config: fc}
		writer := s3Writer{config: s3}

		s := service{
			reader: reader,
			writer: writer,
		}
		factsetRes := getResourceList(*resources)

		c := cron.New()
		//run the upload every monday at 1:00 PM
		c.AddFunc("0 0 13 * * 1", func() {
			err := s.UploadFromFactset(factsetRes)
			if err != nil {
				log.Error(err)
			}
		})
		c.Start()

		httpHandler := &httpHandler{}
		listen(httpHandler, *port)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("[%v]", err)
	}
}

func getResourceList(resources string) []factsetResource {
	factsetRes := []factsetResource{}
	resList := strings.Split(resources, serSeparator)
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
	err := http.ListenAndServe(":"+strconv.Itoa(port), r)
	if err != nil {
		log.Error(err)
	}
}
