package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"strings"
)

const testFolder = "test"
var filesToRead = []string{"edm_security_entity_map.txt"}

func TestFactsetReader_GetLastVersion(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	fim := []fileInfoMock{
		{
			name: "edm_premium_v2_full_1532.zip",
		},
		{
			name: "edm_premium_v1_full_1547.zip",
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
			res:      "edm_premium",
			files:    fis,
			expected: "edm_premium_v2_full_1532.zip",
		},
	}

	for _, tc := range tcs {
		lastVers, err := fsReader.getLastVersion(tc.files, tc.res)
		as.NoError(err)
		as.Equal(tc.expected, lastVers)
	}
}

func TestFactsetReader_GetLastVersion_NoMatch(t *testing.T) {
	as := assert.New(t)
	fsReader := FactsetReader{}

	fim := []fileInfoMock{
		{
			name: "edm_premium_v1_full_1547.zip",
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
		res:   "sym_premium",
		files: fis,
	}

	lastVers, err := fsReader.getLastVersion(tcs.files, tcs.res)
	as.Error(err)
	as.Equal(lastVers, "")
}

func TestFactsetReader_GetLastVersion_ConversionError(t *testing.T) {
	as := assert.New(t)
	fsReader := FactsetReader{}

	fim := []fileInfoMock{
		{
			name: "edm_premium_v1_full_noMinorVersion.zip",
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
		res:   "edm_premium",
		files: fis,
	}

	lastVers, err := fsReader.getLastVersion(tcs.files, tcs.res)
	as.Error(err)
	as.Equal(lastVers, "")
}

func TestFactsetReader_Unzip(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tc := struct {
		archive string
		names    []string
		dest    string
	}{
		archive: "edm_premium_v1_full_1532.zip",
		names:    filesToRead,
		dest:    testFolder,
	}

	for _, name := range tc.names {
		err := fsReader.unzip(tc.archive, name, tc.dest)
		as.NoError(err)

		fileName := path.Join(tc.dest, name)
		file, err := os.Open(fileName)
		as.NotNil(file)
		as.NoError(err)
		file.Close()
		defer as.NoError(os.Remove(fileName))

	}
}

func TestFactsetReader_Unzip_ReaderError(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tc := struct {
		archive string
		names    []string
		dest    string
	}{
		archive: "sample_v1_full_1532.zip",
		names:    append(filesToRead,"sample_entity_map.txt"),
		dest:    testFolder,
	}

	for _, name := range tc.names {
		err := fsReader.unzip(tc.archive, name, tc.dest)
		as.Error(err)
	}
}

func TestFactsetReader_Unzip_NoMatch(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tc := struct {
		archive string
		names    []string
		dest    string
	}{
		archive: "edm_premium_v1_full_1532.zip",
		names:    append(filesToRead, "sample_map.txt"),
		dest:    testFolder,
	}

	for _, name := range tc.names {
		err := fsReader.unzip(tc.archive, name, tc.dest)
		as.Nil(err)
	}
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
		fileName: "edm_premium_v1_full_1532.zip",
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
		readDirMock: getReadDirMock([]string{"edm_premium_v1_full_1532.zip", "edm_premium_v1_full_1522.zip"}),
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
		archive:  "test/edm_premium",
		fileNames: "edm_security_entity_map.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	f, err := fsReader.Read(factsetRes, dest)
	as.NoError(err)
	as.Equal(f, "edm_premium_v1_full_1532.zip")

	files := strings.Split(factsetRes.fileNames, ";")
	for _, fileName := range files {
		file, err := os.Open(path.Join(dest, fileName))
		as.NoError(err)
		file.Close()

		defer as.NoError(os.RemoveAll(dest))
	}
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
		archive:  "test/edm_premium",
		fileNames: "edm_security_entity_map.txt;edm_entities.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest)
	as.Error(err)
}

func TestFactsetReader_Read_DownloadError(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: getReadDirMock([]string{"edm_premium_v1_full_1532.zip", "edm_premium_v1_full_1522.zip"}),
		downloadMock: func(fileName string, dest string) error {
			return fmt.Errorf("Could not download file [%s] from [%s]", fileName, dest)
		},
	}

	fsReader := FactsetReader{client: &sftpClient}

	factsetRes := factsetResource{
		archive:  "test/edm_premium",
		fileNames: "edm_security_entity_map.txt;edm_entities.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest)
	as.Error(err)
}

func TestFactsetReader_Read_GetLastVersionError(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: getReadDirMock([]string{"edm_premium_v1_full_9823372036854775808.zip", "edm_premium_v1_full_1522.zip"}),
		downloadMock: func(fileName string, dest string) error {
			return nil
		},
	}

	fsReader := FactsetReader{client: &sftpClient}

	factsetRes := factsetResource{
		archive:  "test/edm_premium",
		fileNames: "edm_security_entity_map.txt;edm_entities.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest)
	as.Error(err)
}

func TestFactsetReader_GetFullVersion(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tcs := []struct {
		name            string
		expectedVersion string
	}{
		{
			name:            "abc_v1_full_2145.zip",
			expectedVersion: "v1_full_2145",
		},
		{
			name:            "abc_v2_full_2445.zip",
			expectedVersion: "v2_full_2445",
		},
	}

	for _, tc := range tcs {
		resultedVersion, err := fsReader.GetFullVersion(tc.name)
		as.NoError(err)
		as.Equal(tc.expectedVersion, resultedVersion)
	}
}

func TestFactsetReader_GetFullVersionError(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tcs := []struct {
		name            string
		expectedVersion string
	}{
		{
			name:            "abc_v1_full.zip",
			expectedVersion: "",
		},
		{
			name:            "abc_full_2445.zip",
			expectedVersion: "",
		},
		{
			name:            "abc_v1_2345.zip",
			expectedVersion: "",
		},
		{
			name:            "abc.zip",
			expectedVersion: "",
		},
	}

	for _, tc := range tcs {
		resultedVersion, err := fsReader.GetFullVersion(tc.name)
		as.Error(err)
		as.Equal(tc.expectedVersion, resultedVersion)
	}
}

func TestFactsetReader_getMajorVersion(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tcs := []struct {
		name            string
		expectedVersion int
	}{
		{
			name:            "v1_full_2145",
			expectedVersion: 1,
		},
		{
			name:            "v2_full_2445",
			expectedVersion: 2,
		},
		{
			name:            "v23_full_2445",
			expectedVersion: 23,
		},
	}

	for _, tc := range tcs {
		resultedVersion, err := fsReader.getMajorVersion(tc.name)
		as.NoError(err)
		as.Equal(tc.expectedVersion, resultedVersion)
	}
}

func TestFactsetReader_getMajorVersionError(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tcs := []struct {
		name            string
		expectedVersion int
	}{
		{
			name:            "full_2233",
			expectedVersion: -1,
		},
		{
			name:            "notAMahorVersion_full_2233",
			expectedVersion: -1,
		},
		{
			name:            "vABC_full_2233",
			expectedVersion: -1,
		},
		{
			name:            "",
			expectedVersion: -1,
		},
	}

	for _, tc := range tcs {
		resultedVersion, err := fsReader.getMajorVersion(tc.name)
		as.Error(err)
		as.Equal(tc.expectedVersion, resultedVersion)
	}
}

func TestFactsetReader_getMinorVersion(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tcs := []struct {
		name            string
		expectedVersion int
	}{
		{
			name:            "v1_full_2145",
			expectedVersion: 2145,
		},
		{
			name:            "v2_full_2445",
			expectedVersion: 2445,
		},
	}

	for _, tc := range tcs {
		resultedVersion, err := fsReader.getMinorVersion(tc.name)
		as.NoError(err)
		as.Equal(tc.expectedVersion, resultedVersion)
	}
}

func TestFactsetReader_getMinorVersionError(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tcs := []struct {
		name            string
		expectedVersion int
	}{
		{
			name:            "v1_full",
			expectedVersion: -1,
		},
		{
			name:            "full_notAMinorVersion",
			expectedVersion: -1,
		},
		{
			name:            "",
			expectedVersion: -1,
		},
	}

	for _, tc := range tcs {
		resultedVersion, err := fsReader.getMinorVersion(tc.name)
		as.Error(err)
		as.Equal(tc.expectedVersion, resultedVersion)
	}
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
