package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"strings"
	"testing"
)

func TestSortAndZipFilesCreatesOneDailyAndOneWeeklyZipWhenValid(t *testing.T) {
	as := assert.New(t)

	ts := service{}
	weeklyCollection := zipCollection{"weekly_files_full.zip", []string{"ppl_people.txt", "edm_entity.txt"}}
	dailyCollection := zipCollection{"daily_files.zip", []string{"ppl_people_update.txt", "ppl_people_delete.txt", "edm_entity_update.txt", "edm_entity_delete.txt"}}
	createTestDirectoriesAndFiles(weeklyCollection)
	createTestDirectoriesAndFiles(dailyCollection)

	zipColls := []zipCollection{weeklyCollection, dailyCollection}

	filesToWrite, err := ts.sortAndZipFiles(zipColls)
	assert.Equal(t, []string{"daily.zip", "weekly.zip"}, filesToWrite)
	as.NoError(err)

	defer removeCreatedDirectoriesAndFiles()
}

func TestSortAndZipFilesCreatesOneWeeklyZipWhenValid(t *testing.T) {
	as := assert.New(t)

	ts := service{weekly: true}
	weeklyCollection := zipCollection{"weekly_files_full.zip", []string{"ppl_people.txt", "edm_entity.txt"}}
	createTestDirectoriesAndFiles(weeklyCollection)

	zipColls := []zipCollection{weeklyCollection}

	filesToWrite, err := ts.sortAndZipFiles(zipColls)
	assert.Equal(t, []string{"weekly.zip"}, filesToWrite)
	as.NoError(err)

	defer removeCreatedDirectoriesAndFiles()
}

func TestSortAndZipFilesCreatesOneDailyZipWhenValid(t *testing.T) {
	as := assert.New(t)

	ts := service{}
	dailyCollection := zipCollection{"daily_files.zip", []string{"ppl_people_update.txt", "ppl_people_delete.txt", "edm_entity_update.txt", "edm_entity_delete.txt"}}
	createTestDirectoriesAndFiles(dailyCollection)

	zipColls := []zipCollection{dailyCollection}

	filesToWrite, err := ts.sortAndZipFiles(zipColls)
	assert.Equal(t, []string{"daily.zip"}, filesToWrite)
	as.NoError(err)

	defer removeCreatedDirectoriesAndFiles()
}

func TestSortAndZipFilesReturnsErrorWhenCollectionIsEmpty(t *testing.T) {
	as := assert.New(t)

	ts := service{}
	emptyCollection := zipCollection{}

	zipColls := []zipCollection{emptyCollection}

	filesToWrite, err := ts.sortAndZipFiles(zipColls)
	assert.Equal(t, []string(nil), filesToWrite)
	as.Error(err)
}

func TestCleanUpOfWorkingDirectory(t *testing.T) {
	as := assert.New(t)

	ts := service{}
	weeklyCollection := zipCollection{"weekly_files_full.zip", []string{"ppl_people.txt", "edm_entity.txt"}}
	dailyCollection := zipCollection{"daily_files.zip", []string{"ppl_people_update.txt", "ppl_people_delete.txt", "edm_entity_update.txt", "edm_entity_delete.txt"}}
	createTestDirectoriesAndFiles(dailyCollection)
	createTestDirectoriesAndFiles(weeklyCollection)

	zipColls := []zipCollection{dailyCollection, weeklyCollection}

	filesToWrite, _ := ts.sortAndZipFiles(zipColls)
	dailyStat, err := os.Stat(dataFolder + "/daily")
	as.NoError(err)
	weeklyStat, err := os.Stat(dataFolder + "/weekly")
	as.NoError(err)
	assert.True(t, dailyStat.IsDir(), "/daily directory should exist with files as it hasnt been cleaned up yet")
	assert.True(t, weeklyStat.IsDir(), "/weekly directory should exist with files as it hasnt been cleaned up yet")

	ts.cleanUpWorkingDirectory([]zipCollection{dailyCollection, weeklyCollection}, filesToWrite)
	_, dailyErr := os.Stat(dataFolder + "/daily")
	as.True(os.IsNotExist(dailyErr))
	_, weeklyErr := os.Stat(dataFolder + "/weekly")
	as.True(os.IsNotExist(weeklyErr))
}

func createTestDirectoriesAndFiles(zc zipCollection) {
	var archive string
	if strings.Contains(zc.archive, "full") {
		archive = "weekly"
	} else {
		archive = "daily"
	}
	os.Mkdir(dataFolder+"/"+archive, 0755)
	for _, file := range zc.filesToWrite {
		createdFile, _ := os.Create(dataFolder + "/" + archive + "/" + file)
		createdFile.Close()
	}

}

func removeCreatedDirectoriesAndFiles() {
	os.RemoveAll(path.Join(dataFolder, weekly))
	os.RemoveAll(path.Join(dataFolder, daily))
	os.Remove(path.Join(dataFolder, "weekly.zip"))
	os.Remove(path.Join(dataFolder, "daily.zip"))
}
