package main

import (
	"os"

	"net/http"
	"strconv"
	"strings"

	"github.com/Financial-Times/go-fthealth/v1a"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jasonlvhit/gocron"
	"github.com/jawher/mow.cli"
)

const resSeparator = ","

var daysSchedulers = map[int]func(j *gocron.Job) *gocron.Job{
	0: func(j *gocron.Job) *gocron.Job { return j.Sunday() },
	1: func(j *gocron.Job) *gocron.Job { return j.Monday() },
	2: func(j *gocron.Job) *gocron.Job { return j.Tuesday() },
	3: func(j *gocron.Job) *gocron.Job { return j.Wednesday() },
	4: func(j *gocron.Job) *gocron.Job { return j.Thursday() },
	5: func(j *gocron.Job) *gocron.Job { return j.Friday() },
	6: func(j *gocron.Job) *gocron.Job { return j.Saturday() },
}

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
		Value:  "/datafeeds/symbology/sym_hub/sym_hub:sym_coverage.txt,/datafeeds/symbology/sym_bbg/sym_bbg:sym_bbg.txt,/datafeeds/symbology/sym_sec_entity/sym_sec_entity:sym_sec_entity.txt,/datafeeds/entity/ent_entity_advanced/ent_entity_advanced:ent_entity_coverage.txt,/datafeeds/reference/ref_hub/ref_hub_v2:ppl_job_function_map.txt,/datafeeds/people/ppl_premium/ppl_premium_v1:ppl_people.txt;ppl_jobs.txt;ppl_job_functions.txt;ppl_titles.txt",
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
		dayOfWeekScheduler := daysSchedulers[weekDay]
		j = dayOfWeekScheduler(j)
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
				fileNames: resPath[1],
			}
			factsetRes = append(factsetRes, fr)

			//files := strings.Split(resPath[1], ";")
			//filesToRead := []string{}
			//for _, file := range files {
			//	filesToRead = append(filesToRead, file)
			//}
			//fr := factsetResource{
			//	archive:  resPath[0],
			//	fileNames: filesToRead,
			//}
			//factsetRes = append(factsetRes, fr)
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
	err := http.ListenAndServe(":"+strconv.Itoa(port), r)
	if err != nil {
		log.Error(err)
	}
}
