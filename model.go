package main

const dataFolder = "data"

type factsetResource struct {
	archive  string
	fileName string
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
	keyPath  string
}
