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
const isWeekly = false

var filesToRead = []string{"edm_security_entity_map.txt"}

func TestFactsetReader_getMinorAndMajorVersions(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tcs := []struct {
		name                 string
		expectedMinorVersion int
		expectedMajorVersion int
	}{
		{
			name:            "v1_full_2145",
			expectedMinorVersion: 2145,
			expectedMajorVersion: 1,
		},
		{
			name:                 "v2_full_2445",
			expectedMinorVersion: 2445,
			expectedMajorVersion: 2,
		},
	}

	for _, tc := range tcs {
		foundMinorVersion, err := fsReader.getMinorVersion(tc.name)
		foundMajorVersion, err := fsReader.getMajorVersion(tc.name)
		as.NoError(err)
		as.Equal(tc.expectedMinorVersion, foundMinorVersion)
		as.Equal(tc.expectedMajorVersion, foundMajorVersion)
	}
}

func TestFactsetReader_getVersionError(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tcs := []struct {
		name                 string
		expectedMinorVersion int
		expectedMajorVersion int
	}{
		{
			name:            "v_full",
			expectedMinorVersion: -1,
			expectedMajorVersion: -1,
		},
		{
			name:                "full_notAMinorVersion",
			expectedMinorVersion: -1,
			expectedMajorVersion: -1,
		},
		{
			name:                 "",
			expectedMinorVersion: -1,
			expectedMajorVersion: -1,
		},
	}

	for _, tc := range tcs {
		foundMinorVersion, err := fsReader.getMinorVersion(tc.name)
		foundMajorVersion, err := fsReader.getMinorVersion(tc.name)
		as.Error(err)
		as.Equal(tc.expectedMinorVersion, foundMinorVersion)
		as.Equal(tc.expectedMajorVersion, foundMajorVersion)
	}
}

func TestGetMostRecentZipsWillReturnBothDailyAndWeeklyZipsIfTheyAreMostRecent(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	fim := []fileInfoMock{
		{
			name: "edm_premium_v1_full_1532.zip",
		},
		{
			name: "edm_premium_v1_full_1547.zip",
		},
		{
			name: "edm_premium_v1_full_1546.zip",
		},
		{
			name: "edm_premium_v1_full_1.zip",
		},
		{
			name: "edm_premium_v1_1547.zip",
		},
		{
			name: "edm_premium_v1_full_1546.zip",
		},
	}

	fis := []os.FileInfo{}
	for _, fi := range fim {
		fis = append(fis, os.FileInfo(fi))
	}

	tcs := []struct {
		res      string
		files    []os.FileInfo
		expected []string
	}{
		{
			res:      "edm_premium",
			files:    fis,
			expected: []string{"edm_premium_v1_full_1547.zip", "edm_premium_v1_1547.zip"},
		},
	}

	for _, tc := range tcs {
		lastVers, err := fsReader.GetMostRecentZips(tc.files, tc.res)
		as.NoError(err)
		as.Equal(tc.expected, lastVers)
	}
}

func TestGetMostRecentZipsPrioritizesMajorVersion(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	fim := []fileInfoMock{
		{
			name: "edm_premium_v1_full_1532.zip",
		},
		{
			name: "edm_premium_v1_full_1547.zip",
		},
		{
			name: "edm_premium_v2_full_1546.zip",
		},
		{
			name: "edm_premium_v1_full_1.zip",
		},
		{
			name: "edm_premium_v1_1547.zip",
		},
		{
			name: "edm_premium_v1_full_1546.zip",
		},
	}

	fis := []os.FileInfo{}
	for _, fi := range fim {
		fis = append(fis, os.FileInfo(fi))
	}

	tcs := []struct {
		res      string
		files    []os.FileInfo
		expected []string
	}{
		{
			res:      "edm_premium",
			files:    fis,
			expected: []string{"edm_premium_v2_full_1546.zip"},
		},
	}

	for _, tc := range tcs {
		lastVers, err := fsReader.GetMostRecentZips(tc.files, tc.res)
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

	lastVers, err := fsReader.GetMostRecentZips(tcs.files, tcs.res)
	as.Error(err)
	as.Empty(lastVers)
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

	lastVers, err := fsReader.GetMostRecentZips(tcs.files, tcs.res)
	as.Error(err)
	as.Empty(lastVers)
	as.Error(err)
}

func TestFactsetReader_GetLastVersionError(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: getReadDirMock([]string{"edm_premium_v1_full_9823372036854775808.zip", "edm_premium_v1_full_1522.zip"}),
		downloadMock: func(fileName string, dest string) error {
			return nil
		},
	}

	fsReader := FactsetReader{client: &sftpClient}

	factsetRes := factsetResource{
		archive:   "test/edm_premium",
		fileNames: "edm_security_entity_map.txt;edm_entities.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest, isWeekly)
	as.Error(err)
}

func TestFactsetReader_Unzip(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	os.Mkdir(testFolder + "/" + weekly, 0755)

	tc := struct {
		archive   string
		names     []string
		dest      string
		subFolder string
	}{
		archive:   "edm_premium_v1_full_1532.zip",
		names:     filesToRead,
		dest:      testFolder,
		subFolder: "/weekly",
	}

	_, err := fsReader.unzip(tc.archive, tc.names, tc.dest)
	as.NoError(err)
	for _, name := range tc.names {
		fileName := path.Join(tc.dest + tc.subFolder, name)
		file, err := os.Open(fileName)
		as.NotNil(file)
		as.NoError(err)
		file.Close()
		as.NoError(os.Remove(tc.dest + tc.subFolder + "/" + name))
	}

}

func TestFactsetReader_Unzip_ReaderError(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tc := struct {
		archive string
		names   []string
		dest    string
	}{
		archive: "sample_v1_full_1532.zip",
		names:   append(filesToRead, "sample_entity_map"),
		dest:    testFolder,
	}

	_, err := fsReader.unzip(tc.archive, tc.names, tc.dest)
	as.Error(err)

}

func TestFactsetReader_Unzip_NoMatch(t *testing.T) {
	as := assert.New(t)

	fsReader := FactsetReader{}

	tc := struct {
		archive string
		names   []string
		dest    string
	}{
		archive: "edm_premium_v1_full_1532.zip",
		names:   append(filesToRead, "sample_map"),
		dest:    testFolder,
	}

	_, err := fsReader.unzip(tc.archive, tc.names, tc.dest)
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
		fileName: "edm_premium_v1_full_1532.zip",
	}

	err := fsReader.download(tc.path, tc.fileName, path.Join(testFolder, dataFolder))
	as.NoError(err)

	file, err := os.Open(path.Join(testFolder, dataFolder, tc.fileName))
	as.NoError(err)
	file.Close()
	as.NoError(os.RemoveAll(path.Join(testFolder, dataFolder)))
}

func TestFactsetReader_ReadRes(t *testing.T) {
	as := assert.New(t)

	sftpClient := sftpClientMock{
		readDirMock: getReadDirMock([]string{"edm_premium_v1_full_1532.zip", "edm_premium_v1_1532.zip", "edm_premium_v1_full_1522.zip"}),
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
		archive:   "test/edm_premium",
		fileNames: "edm_security_entity_map.txt",
	}
	dest := path.Join(testFolder, "/weekly")
	zipColls, err := fsReader.Read(factsetRes, dest, isWeekly)
	for _, zipColl := range zipColls {
		as.NoError(err)
		as.True(strings.Contains(zipColl.archive, "1532"))
		as.Equal("edm_premium_v1_full_1532.zip", zipColl.archive)
	}
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
		archive:   "test/edm_premium",
		fileNames: "edm_security_entity_map.txt;edm_entities",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest, isWeekly)
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
		archive:   "test/edm_premium",
		fileNames: "edm_security_entity_map.txt;edm_entities.txt",
	}
	dest := path.Join(testFolder, dataFolder)
	_, err := fsReader.Read(factsetRes, dest, isWeekly)
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
