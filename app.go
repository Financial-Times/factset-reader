package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
)

//const bbgIDs = "edm_bbg_ids.txt"
const securityEntityMap = "edm_security_entity_map.txt"

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
		Value:  "fts.factset.com",
		Desc:   "factset ftp server address",
		EnvVar: "FACTSET_FTP",
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
		err := s.UploadFromFactset([]factsetResource{
			{
				archive:  "/datafeeds/edm/edm_premium/edm_premium_full",
				fileName: securityEntityMap,
			},
			//{
			//	archive:  "/datafeeds/edm/edm_bbg_ids",
			//	fileName: bbgIDs,
			//},
		})

		if err != nil {
			log.Error(err)
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("[%v]", err)
	}
}
