package main

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"github.com/pkg/errors"
	"fmt"
)

type factsetClient interface {
	Init() error
	Close()
	ReadDir(dir string) ([]os.FileInfo, error)
	Download(path string, dest string) error
}

type sftpClient struct {
	config sftpConfig
	ssh    *ssh.Client
	sftp   *sftp.Client
}

func (s *sftpClient) getSSHConfig(keyPath string, username string) (*ssh.ClientConfig, error) {
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return &ssh.ClientConfig{}, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return &ssh.ClientConfig{}, err
	}

	c := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}
	return c, nil
}

func (s *sftpClient) initSSHClient(config sftpConfig) error {
	c, err := s.getSSHConfig(s.config.keyPath, s.config.username)
	if err != nil {
		return err
	}

	tcpConn, err := ssh.Dial("tcp", config.address + ":" + strconv.Itoa(config.port), c)
	if err != nil {
		return err
	}

	s.ssh = tcpConn
	return nil
}

func (s *sftpClient) Init() error {
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

func (s *sftpClient) ReadDir(dir string) ([]os.FileInfo, error) {
	return s.sftp.ReadDir(dir)
}

func (s sftpClient) Download(path string, dest string) error {
	file, err := s.sftp.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return s.save(file, dest)
}

func (s *sftpClient) save(file *sftp.File, dest string) error {
	os.Mkdir(dest, 0755)
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
		errMsg := fmt.Sprintf("Download stopped at [%d]", n)
		return errors.New(errMsg)
	}
	if err != nil {
		return err
	}

	return nil
}

func (s *sftpClient) Close() {
	s.ssh.Close()
	s.sftp.Close()
}
