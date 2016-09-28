package main

import (
	"os"

	"net/http"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jasonlvhit/gocron"
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
		Value:  "/datafeeds/edm/edm_premium/edm_premium_full:edm_security_entity_map.txt,/datafeeds/edm/edm_bbg_ids/edm_bbg_ids_v1_full:edm_bbg_ids.txt",
		Desc:   "factset resources to be loaded",
		EnvVar: "FACTSET_RESOURCES",
	})

	runningTime := app.String(cli.StringOpt{
		Name:   "runningTime",
		Value:  "1 12 00", // default run the job every Monday at 12:00 PM
		Desc:   "Time at which the job will be run",
		EnvVar: "RUNNING_TIME",
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
			sch := gocron.NewScheduler()
			schedule(sch, *runningTime, func() {
				s.Fetch()
			})
			<-sch.Start()
		}()

		httpHandler := &httpHandler{s: s}
		listen(httpHandler, *port)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("[%v]", err)
	}
}

func schedule(scheduler *gocron.Scheduler, time string, job func()) {
	timeVars := strings.Split(time, " ")
	if len(timeVars) == 3 {
		weekDay, err := strconv.Atoi(timeVars[0])
		if err != nil {
			log.Errorf("Cannot parse running time [%s]", time)
		}

		runningTime := timeVars[1] + ":" + timeVars[2]
		var j *gocron.Job
		j = scheduler.Every(1)
		switch weekDay {
		case 0:
			j = j.Sunday()
			break
		case 1:
			j = j.Monday()
			break
		case 2:
			j = j.Tuesday()
			break
		case 3:
			j = j.Wednesday()
			break
		case 4:
			j = j.Thursday()
			break
		case 5:
			j = j.Friday()
			break
		case 6:
			j = j.Saturday()
			break
		default:
			log.Errorf("Cannot parse running time [%s]", time)
		}
		j.At(runningTime).Do(job)
	} else {
		scheduler.Every(1).Monday().At("12:00").Do(job)
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
	r.HandleFunc("/force-import", h.s.forceImport).Methods("POST")
	err := http.ListenAndServe(":"+strconv.Itoa(port), r)
	if err != nil {
		log.Error(err)
	}
}
