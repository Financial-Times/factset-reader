package main

const dataFolder = "data"

type factsetResource struct {
	archive  string
	fileNames string
}

type s3Config struct {
	accKey    string
	secretKey string
	bucket    string
	domain    string
}

type sftpConfig struct {
	address  string
	port     int
	username string
	key      string
}
