package main

import (
	"archive/zip"
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type reader interface {
	ReadRes(fRes factsetResource) error
}

type factsetReader struct {
	config factsetConfig
}

const pathSeparator = "/"

func (sfr factsetReader) ReadRes(fRes factsetResource) error {
	var auths []ssh.AuthMethod

	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))

	if err != nil {
		return err
	}

	auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
	//auths = append(auths, ssh.Password(sfr.config.password))

	c := ssh.ClientConfig{
		User: sfr.config.username,
		Auth: auths,
	}

	tcpConn, err := ssh.Dial("tcp", sfr.config.address+":22", &c)
	if err != nil {
		return err
	}
	defer tcpConn.Close()

	client, err := sftp.NewClient(tcpConn)
	if err != nil {
		return err
	}
	defer client.Close()

	dir, lastArch, err := getLastVersionOfArch(client, fRes.archive)

	err = downloadArch(client, dir, lastArch, dataFolder)
	if err != nil {
		return err
	}

	err = copyFileFromArch(lastArch, fRes.fileName, dataFolder)
	return err
}

func getLastVersionOfArch(client *sftp.Client, path string) (string, string, error) {
	pathVars := strings.Split(path, pathSeparator)

	dir := ""
	for i := 0; i < len(pathVars)-1; i++ {
		dir += pathVars[i] + pathSeparator
	}
	files, err := client.ReadDir(dir)
	if err != nil {
		return "", "", err
	}

	lastFile := &struct {
		name         string
		lastModified time.Time
	}{}

	for _, file := range files {
		name := file.Name()
		if strings.Contains(name, pathVars[len(pathVars)-1]) {
			if lastFile == nil {
				lastFile.name = name
				lastFile.lastModified = file.ModTime()
			} else {
				if file.ModTime().After(lastFile.lastModified) {
					lastFile.name = name
					lastFile.lastModified = file.ModTime()
				}
			}
		}
	}
	return dir, lastFile.name, nil
}

func downloadArch(client *sftp.Client, path string, name string, dataFolder string) error {
	os.Mkdir(dataFolder, 0755)
	downFile, err := os.Create(dataFolder + pathSeparator + name)
	if err != nil {
		return err
	}
	defer downFile.Close()

	log.Infof("Starting downloading file [%s] from [%s]", name, path)

	r, err := client.Open(path + name)
	if err != nil {
		return err
	}
	defer r.Close()

	const size int64 = 1e9

	n, err := io.Copy(downFile, io.LimitReader(r, size))
	if err != nil {
		return err
	}

	if n != size {
		log.Errorf("")
	}
	log.Infof("Latest version of [%s] was downloaded successfully", name)
	return nil
}

func copyFileFromArch(archName string, name string, dataFolder string) error {
	r, err := zip.OpenReader(dataFolder + pathSeparator + archName)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if name == f.Name {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			file, err := os.Create(dataFolder + pathSeparator + f.Name)
			if err != nil {
				return err
			}
			_, err = io.Copy(file, rc)
			if err != nil {
				return err
			}
			rc.Close()
		}
	}
	return nil
}
