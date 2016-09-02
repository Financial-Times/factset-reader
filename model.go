package main

const dataFolder = "data1"

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

type factsetConfig struct {
	address  string
	username string
	password string
}
