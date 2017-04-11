# Factset Reader (factset-reader)

[![Circle CI](https://circleci.com/gh/Financial-Times/factset-reader/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/factset-reader/tree/master) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/factset-reader/badge.svg)](https://coveralls.io/github/Financial-Times/factset-reader)

__A service for reading files from Factset FTP (SFTP) server and writing them into an Amazon S3 bucket__

# Installation

1. go get github.com/kardianos/govendor
2. go get github.com/Financial-Times/factset-reader
3. cd $GOPATH/src/github.com/Financial-Times/factset-reader
4. govendor sync

# Running

```
$GOPATH/bin/factset-reader
--awsAccessKey=xxx
--awsSecretKey=xxx
--bucketName=com.ft.coco-factset-data
--s3Domain=s3.amazonaws.com
--port=8080
--factsetUser=xxx
--factsetKey=xxx
--factsetFTP=fts-sftp.factset.com
--factsetPort=6671
--resources=/directory/without/version:fileToDownload1.txt;fileToDownload2.txt
```

The awsAccessKey, awsSecretKey, bucketName, factsetUser, factsetKey arguments are mandatory, and represent authentication credentials for S3 and Factset FTP server. 

The resources argument specifies a comma separated list of archives and files within that archive to be downloaded from Factset FTP server. Because every file is inside an archive, the service will first download the archive, unzip the files you specify, zip a collection of daily/weekly files and upload the resulting zips to s3. A resource has the format archive_path:file1.txt;file2.txt, example: /datafeeds/edm/edm_bbg_ids/edm_bbg_ids:edm_bbg_ids.txt, where  /datafeeds/edm/edm_bbg_ids/ is the path of the archive, edm_bbg_ids is the prefix of the zip without versions and edm_bbg_ids.txt is the file to be extracted from this archive. On the Factset FTP server the archive name will contain also the data version, but it is enough for this service to provide the archive name without the version and it will download the latest one.

After downloading the zip files from Factset FTP server, the service will write them to the specified Amazon S3 bucket. The zip files written to S3 will be inside of a folder named by the current date. Depending upon the day there may be both a weekly.zip and daily.zip or just a daily.zip

# Endpoints

Force-import (initiate importing manually of all most recent files):

`http://localhost:8080/force-import -XPOST`

Force-import-weekly (initiate importing manually of all most recent weekly files):

`http://localhost:8080/force-import-weekly -XPOST`

## Admin Endpoints
Health checks: `http://localhost:8080/__health`

Good to go: `http://localhost:8080/__gtg`
