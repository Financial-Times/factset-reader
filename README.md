# Factset Reader (factset-reader)

__A service for reading files from Factset FTP (SFTP) server and writing them into an Amazon S3 bucket.__
# Installation

For the first time:

`go get github.com/Financial-Times/factset-reader`

or update:

`go get -u github.com/Financial-Times/factset-reader`

# Running

`$GOPATH/bin/factset-reader --awsAccessKey=xxx --awsSecretKey=xxx --bucketName=com.ft.coco-factset-data --s3Domain=s3.amazonaws.com --port=8080 --factsetUser=xxx --factsetKey=xxx --factsetFTP=fts-sftp.factset.com --factsetPort=6671 --resources=/datafeeds/edm/edm_premium/edm_premium_full:edm_security_entity_map.txt,/datafeeds/edm/edm_bbg_ids/edm_bbg_ids_v1_full:edm_bbg_ids.txt --runningTime="1 12 00"`

The awsAccessKey, awsSecretKey, bucketName, factsetUser, factsetKey arguments are mandatory, and represent authentication credentials for S3 and Factset FTP server. The other arguments are optional and they will default at reading the edm_security_entity_map.txt and edm_bbg_ids.txt files from Factset and writting them to S3, every Monday at 12:00 PM.

The resources argument specifies a comma separated list of files to be downloaded from Factset FTP server. Because every file is inside an archive, the service will first download the archive, unzip the file and write it to S3 bucket. A resource has the format archive_path:file, example: /datafeeds/edm/edm_bbg_ids/edm_bbg_ids_v1_full:edm_bbg_ids.txt, where  /datafeeds/edm/edm_bbg_ids/edm_bbg_ids_v1_full is the path of the archive without version and edm_bbg_ids.txt is the file to be extracted from this archive. On the Factset FTP server the archive name will contain also the data version, but it is enough for this service to provide the archive name without the version and it will download the latest one.

The runningTime argument specifies when the job should run. It has the format "day_of_week hour minute". Example: "1 12 OO" will run on every Monday at 12:00 PM. The day_of_week parameter takes values from 0 to 6, corresponding to Sunday-Saturday.

After downloading the files from Factset FTP server, the service will write them to the specified Amamzon S3 bucket. The file written to S3 will have as name the original name of the file appended with the current date.

# Endpoints

Force-import (initiate importing manually): http://localhost:8080/force-import -XPOST

## Admin Endpoints
Health checks: http://localhost:8080/__health

Good to go: http://localhost:8080/__gtg
