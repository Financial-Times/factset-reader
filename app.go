package main

import (
	"os"

	"net/http"
	"strconv"
	"strings"

	"github.com/Financial-Times/go-fthealth/v1a"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
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
		Value:  "",
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
			key:      *factsetKey,
			port:     *factsetPort,
		}

		s := service{
			rdConfig: fc,
			wrConfig: s3,
			files:    getResourceList(*resources),
		}

		log.Printf("Resource list: %v", s.files)
		go func() {
			s.fetchResources(s.files)
		}()

		httpHandler := &httpHandler{s: s}
		listen(httpHandler, *port)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("[%v]", err)
	}
}

func getResourceList(resources string) []factsetResource {
	if resources == "" {
		return []factsetResource{}
	}
	factsetRes := []factsetResource{}
	resList := strings.Split(resources, resSeparator)
	for _, fulRes := range resList {
		resPath := strings.Split(fulRes, ":")
		if len(resPath) == 2 {
			fr := factsetResource{
				archive:   resPath[0],
				fileNames: resPath[1],
			}
			factsetRes = append(factsetRes, fr)
		}
	}
	return factsetRes
}

func listen(h *httpHandler, port int) {
	log.Infof("Listening on port: %d", port)
	r := mux.NewRouter()
	r.HandleFunc("/__health", v1a.Handler("Factset Reader Healthchecks", "Checks for accessing Factset server and Amazon S3 bucket", h.factsetHealthcheck(), h.amazonS3Healthcheck()))
	r.HandleFunc("/__gtg", h.goodToGo)
	r.HandleFunc("/force-import", h.s.forceImport).Methods("POST")
	r.HandleFunc("/force-import-weekly", h.s.forceImportWeekly).Methods("POST")
	err := http.ListenAndServe(":"+strconv.Itoa(port), r)
	if err != nil {
		log.Error(err)
	}
}
