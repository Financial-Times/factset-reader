package main

const dataFolder = "data"
const weekly = "weekly"
const daily = "daily"

type factsetResource struct {
	archive   string
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

type zipCollection struct {
	archive      string
	filesToWrite []string
}
