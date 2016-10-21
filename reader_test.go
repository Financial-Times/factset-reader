package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testFolder = "test"

func TestFactsetReader_GetLastVersion(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	fim := []fileInfoMock{
		{
			name: "edm_premium_full_1532.zip",
		},
		{
			name: "edm_premium_full_1547.zip",
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
			expected: "edm_premium_full_1547.zip",
		},
	}

	for _, tc := range tcs {
		lastVers, err := fsReader.getLastVersion(tc.files, tc.res)
		as.NoError(err)
		as.Equal(lastVers, tc.expected)
	}
}

func TestFactsetReader_GetLastVersion_NoMatch(t *testing.T) {
	as := assert.New(t)
	fsReader := FactsetReader{}

	fim := []fileInfoMock{
		{
			name: "edm_premium_full_1547.zip",
		},
	}

	fis := []os.FileInfo{}
	for _, fi := range fim {
		fis = append(fis, os.FileInfo(fi))
	}

	tcs := struct {
		res   string
		files []os.FileInfo
	}{
		res:   "sym_premium_full",
		files: fis,
	}

	lastVers, err := fsReader.getLastVersion(tcs.files, tcs.res)
	as.NoError(err)
	as.Equal(lastVers, "")
}

func TestFactsetReader_GetLastVersion_ConversionError(t *testing.T) {
	as := assert.New(t)
	fsReader := FactsetReader{}

	fim := []fileInfoMock{
		{
			name: "edm_premium_full_9823372036854775808.zip",
		},
	}

	fis := []os.FileInfo{}
	for _, fi := range fim {
		fis = append(fis, os.FileInfo(fi))
	}

	tcs := struct {
		res   string
		files []os.FileInfo
	}{
		res:   "edm_premium_full",
		files: fis,
	}

	_, err := fsReader.getLastVersion(tcs.files, tcs.res)
	as.Error(err)
}

func TestFactsetReader_Unzip(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

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

func TestFactsetReader_Unzip_ReaderError(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tc := struct {
		archive string
		name    string
		dest    string
	}{
		archive: "sample_full_1532.zip",
		name:    "sample_entity_map.txt",
		dest:    testFolder,
	}

	err := fsReader.unzip(tc.archive, tc.name, tc.dest)
	as.Error(err)
}

func TestFactsetReader_Unzip_NoMatch(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tc := struct {
		archive string
		name    string
		dest    string
	}{
		archive: "edm_premium_full_1532.zip",
		name:    "sample_map.txt",
		dest:    testFolder,
	}

	err := fsReader.unzip(tc.archive, tc.name, tc.dest)
	as.Nil(err)
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

	fsReader := FactsetReader{client: &sftpClient}

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
		readDirMock: getReadDirMock([]string{"edm_premium_full_1532.zip", "edm_premium_full_1522.zip"}),
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

	fsReader := FactsetReader{client: &sftpClient}

	factsetRes := factsetResource{
		archive:  "test/edm_premium_full",
		fileName: "edm_security_entity_map.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	f, err := fsReader.Read(factsetRes, dest)
	as.NoError(err)
	as.Equal(f, "edm_premium_full_1532.zip")

	file, err := os.Open(path.Join(dest, factsetRes.fileName))
	as.NoError(err)
	file.Close()

	defer as.NoError(os.RemoveAll(dest))
}

func TestFactsetReader_Read_ReadDirErr(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: func(dir string) ([]os.FileInfo, error) {
			return nil, fmt.Errorf("Could not read directory [%s]", dir)
		},
		downloadMock: func(fileName string, dest string) error {
			return nil
		},
	}

	fsReader := FactsetReader{client: &sftpClient}

	factsetRes := factsetResource{
		archive:  "test/edm_premium_full",
		fileName: "edm_security_entity_map.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest)
	as.Error(err)
}

func TestFactsetReader_Read_DownloadError(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: getReadDirMock([]string{"edm_premium_full_1532.zip", "edm_premium_full_1522.zip"}),
		downloadMock: func(fileName string, dest string) error {
			return fmt.Errorf("Could not download file [%s] from [%s]", fileName, dest)
		},
	}

	fsReader := FactsetReader{client: &sftpClient}

	factsetRes := factsetResource{
		archive:  "test/edm_premium_full",
		fileName: "edm_security_entity_map.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest)
	as.Error(err)
}

func TestFactsetReader_Read_GetLastVersionError(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: getReadDirMock([]string{"edm_premium_full_9823372036854775808.zip", "edm_premium_full_1522.zip"}),
		downloadMock: func(fileName string, dest string) error {
			return nil
		},
	}

	fsReader := FactsetReader{client: &sftpClient}

	factsetRes := factsetResource{
		archive:  "test/edm_premium_full",
		fileName: "edm_security_entity_map.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest)
	as.Error(err)
}

func getReadDirMock(files []string) func(dir string) ([]os.FileInfo, error) {
	filesInfo := []fileInfoMock{}
	for _, file := range files {
		filesInfo = append(filesInfo, fileInfoMock{name: file})
	}
	return func(dir string) ([]os.FileInfo, error) {
		fim := filesInfo
		fis := []os.FileInfo{}
		for _, fi := range fim {
			fis = append(fis, os.FileInfo(fi))
		}
		return fis, nil
	}

}
