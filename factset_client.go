package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type FactsetClient interface {
	Init() error
	Close()
	ReadDir(dir string) ([]os.FileInfo, error)
	Download(path string, dest string) error
}

type SFTPClient struct {
	config sftpConfig
	ssh    *ssh.Client
	sftp   *sftp.Client
}

func (s *SFTPClient) getSSHConfig(username string, key string) (*ssh.ClientConfig, error) {
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return &ssh.ClientConfig{}, err
	}

	c := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return c, nil
}

func (s *SFTPClient) initSSHClient(config sftpConfig) error {
	c, err := s.getSSHConfig(s.config.username, s.config.key)
	if err != nil {
		return err
	}

	tcpConn, err := ssh.Dial("tcp", config.address+":"+strconv.Itoa(config.port), c)
	if err != nil {
		return err
	}

	s.ssh = tcpConn
	return nil
}

func (s *SFTPClient) Init() error {
	err := s.initSSHClient(s.config)
	if err != nil {
		return err
	}
	client, err := sftp.NewClient(s.ssh)
	if err != nil {
		return err
	}
	s.sftp = client
	return nil
}

func (s *SFTPClient) ReadDir(dir string) ([]os.FileInfo, error) {
	return s.sftp.ReadDir(dir)
}

func (s SFTPClient) Download(path string, dest string) error {
	file, err := s.sftp.Open(path)
	file.Name()
	if err != nil {
		return err
	}
	defer file.Close()
	return s.save(file, dest)
}

func (s *SFTPClient) save(file *sftp.File, dest string) error {
	os.Mkdir(dest, 0700)
	_, fileName := path.Split(file.Name())
	downFile, err := os.Create(path.Join(dest, fileName))
	if err != nil {
		return err
	}
	defer downFile.Close()

	fileStat, err := file.Stat()
	if err != nil {
		return err
	}
	size := fileStat.Size()

	n, err := io.Copy(downFile, io.LimitReader(file, size))
	if n != size {
		e := fmt.Sprintf("Download stopped at [%d]", n)
		return errors.New(e)
	}
	if err != nil {
		return err
	}

	return nil
}

func (s *SFTPClient) Close() {
	if s.ssh != nil {
		s.ssh.Close()
	}
	if s.sftp != nil {
		s.sftp.Close()
	}
}
