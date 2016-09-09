package main

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

const testFolder = "test"

func TestFactsetReader_GetLastVersion(t *testing.T) {
	as := assert.New(t)

	fsReader := factsetReader{}

	fim := []fileInfoMock{
		{
			name:  "edm_premium_full_1532",
			mtime: time.Date(2016, time.September, 6, 23, 0, 0, 0, time.UTC),
		},
		{
			name:  "edm_premium_full_1547",
			mtime: time.Date(2016, time.September, 7, 23, 0, 0, 0, time.UTC),
		},
	}

	fis := []os.FileInfo{}
	for _, fi := range fim {
		fis = append(fis, os.FileInfo(fi))
	}

	tcs := []struct {
		res      string
		files    []os.FileInfo
		expected string
	}{
		{
			res:      "edm_premium_full",
			files:    fis,
			expected: "edm_premium_full_1547",
		},
	}

	for _, tc := range tcs {
		lastVers, err := fsReader.getLastVersion(tc.files, tc.res)
		as.NoError(err)
		as.Equal(lastVers, tc.expected)
	}
}

func TestFactsetReader_Unzip(t *testing.T) {
	as := assert.New(t)

	fsReader := factsetReader{}

	tc := struct {
		archive string
		name    string
		dest    string
	}{
		archive: "edm_premium_full_1532.zip",
		name:    "edm_security_entity_map.txt",
		dest:    testFolder,
	}

	err := fsReader.unzip(tc.archive, tc.name, tc.dest)
	as.NoError(err)

	fileName := path.Join(tc.dest, tc.name)
	file, err := os.Open(fileName)
	as.NotNil(file)
	as.NoError(err)
	file.Close()
	defer as.NoError(os.Remove(fileName))
}

func TestFactsetReader_Download(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: func(dir string) ([]os.FileInfo, error) {
			return []os.FileInfo{}, nil
		},
		downloadMock: func(fileName string, dest string) error {
			content, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}
			os.Mkdir(dest, 0755)
			_, name := path.Split(fileName)
			err = ioutil.WriteFile(path.Join(dest, name), content, 0644)
			return err
		},
	}

	fsReader := factsetReader{client: &sftpClient}

	tc := struct {
		path     string
		fileName string
	}{
		path:     testFolder,
		fileName: "edm_premium_full_1532.zip",
	}

	err := fsReader.download(tc.path, tc.fileName, path.Join(testFolder, dataFolder))
	as.NoError(err)

	file, err := os.Open(path.Join(testFolder, dataFolder, tc.fileName))
	as.NoError(err)
	file.Close()
	defer as.NoError(os.RemoveAll(path.Join(testFolder, dataFolder)))
}

func TestFactsetReader_ReadRes(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: func(dir string) ([]os.FileInfo, error) {
			fim := []fileInfoMock{
				{
					name:  "edm_premium_full_1532.zip",
					mtime: time.Date(2016, time.September, 6, 23, 0, 0, 0, time.UTC),
				},
				{
					name:  "edm_premium_full_1522.zip",
					mtime: time.Date(2016, time.September, 1, 23, 0, 0, 0, time.UTC),
				},
			}

			fis := []os.FileInfo{}
			for _, fi := range fim {
				fis = append(fis, os.FileInfo(fi))
			}
			return fis, nil
		},
		downloadMock: func(fileName string, dest string) error {
			content, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}
			err = os.Mkdir(dest, 0755)
			if err != nil {
				return err
			}
			_, name := path.Split(fileName)
			err = ioutil.WriteFile(path.Join(dest, name), content, 0644)
			return err
		},
	}

	fsReader := factsetReader{client: &sftpClient}

	factsetRes := factsetResource{
		archive:  "test/edm_premium_full",
		fileName: "edm_security_entity_map.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	err := fsReader.ReadRes(factsetRes, dest)
	as.NoError(err)

	file, err := os.Open(path.Join(dest, factsetRes.fileName))
	as.NoError(err)
	file.Close()

	defer as.NoError(os.RemoveAll(dest))
}
