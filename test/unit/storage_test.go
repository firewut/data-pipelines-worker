package unit_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"data-pipelines-worker/types"
	"data-pipelines-worker/types/interfaces"
)

const (
	minioTestFolder = "test-folder"
)

func (suite *UnitTestSuite) TestDetectMimeTypeFromBuffer() {
	type cases struct {
		content   string
		mimeType  string
		extension string
	}
	cases_list := []cases{
		{textContent, "text/plain; charset=utf-8", ".txt"},
		{xmlContent, "text/xml; charset=utf-8", ".xml"},
	}
	for _, _case := range cases_list {
		buffer := bytes.NewBufferString(_case.content)
		mimeType, err := types.DetectMimeTypeFromBuffer(buffer)
		suite.Nil(err)
		suite.Equal(_case.mimeType, mimeType.String())
		suite.Equal(_case.extension, mimeType.Extension())
	}
}

func (suite *UnitTestSuite) TestLocalStorageListObjects() {
	// Given
	storage := types.NewLocalStorage("")

	// When
	objects, err := storage.ListObjects(
		interfaces.StorageLocation{
			LocalDirectory: os.TempDir(),
		},
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(objects)
}

func (suite *UnitTestSuite) TestLocalStoragePutObject() {
	// Given
	fileContent := bytes.NewBufferString(textContent)
	oldFileName := fmt.Sprintf("%s.txt", uuid.NewString())
	newFileName := fmt.Sprintf("%s.txt", uuid.NewString())
	storage := types.NewLocalStorage("")

	file, err := os.Create(
		filepath.Join(
			os.TempDir(),
			oldFileName,
		),
	)
	suite.Nil(err)
	defer file.Close()
	_, err = file.Write(fileContent.Bytes())
	suite.Nil(err)

	source := interfaces.StorageLocation{
		LocalDirectory: os.TempDir(),
		FileName:       oldFileName,
	}
	destination := interfaces.StorageLocation{
		LocalDirectory: os.TempDir(),
		FileName:       newFileName,
	}

	// When
	err = storage.PutObject(source, destination)

	// Then
	suite.Nil(err)
	defer storage.DeleteObject(destination)
	defer storage.DeleteObject(source)

	// Ensure the new file exists and has the same content
	newFile, err := os.ReadFile(filepath.Join(os.TempDir(), newFileName))
	suite.Nil(err)
	suite.Equal(textContent, string(newFile))
}

func (suite *UnitTestSuite) TestLocalStoragePutObjectBytes() {
	// Given
	storage := types.NewLocalStorage("")

	fileContent := bytes.NewBufferString(textContent)

	// When
	storageLocation, err := storage.PutObjectBytes(
		storage.GetStorageLocation(""),
		fileContent,
	)

	// Then
	suite.Nil(err)
	defer storage.DeleteObject(storageLocation)

	suite.NotEmpty(storageLocation.FileName)
	suite.Contains(storageLocation.FileName, ".txt")
}

func (suite *UnitTestSuite) TestLocalStorageGetObjectBytes() {
	// Given
	storage := types.NewLocalStorage("")
	fileContent := bytes.NewBufferString(xmlContent)

	// When
	storageLocation, err := storage.PutObjectBytes(
		storage.GetStorageLocation(""),
		fileContent,
	)

	// Then
	suite.Nil(err)
	defer storage.DeleteObject(storageLocation)

	suite.NotEmpty(storageLocation.FileName)
	suite.Contains(storageLocation.FileName, ".xml")

	objectBytes, err := storage.GetObjectBytes(storageLocation)
	suite.Nil(err)
	suite.Equal(xmlContent, objectBytes.String())
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObject() {
	// Given
	storage := types.NewMINIOStorage()

	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.GetStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorage.DeleteObject(localStorageLocation)

	suite.NotEmpty(localStorageLocation.FileName)
	suite.Contains(localStorageLocation.FileName, ".txt")

	remoteStorageLocation := storage.GetStorageLocation(
		filepath.Join(
			minioTestFolder,
			localStorageLocation.FileName,
		),
	)

	// When
	err = storage.PutObject(
		localStorageLocation,
		remoteStorageLocation,
	)

	// Then
	suite.Nil(err)
	defer storage.DeleteObject(remoteStorageLocation)
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObjectBytes() {
	// Given
	storage := types.NewMINIOStorage()
	localStorage := types.NewLocalStorage("")
	fileNameWithoutExt := uuid.NewString()

	// When
	localStorageLocation, err := storage.PutObjectBytes(
		storage.GetStorageLocation(
			filepath.Join(minioTestFolder, fileNameWithoutExt),
		),
		bytes.NewBufferString(xmlContent),
	)

	// Then
	suite.Nil(err)
	defer localStorage.DeleteObject(localStorageLocation)
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObjectBytesLongName() {
	// Given
	storage := types.NewMINIOStorage()
	localStorage := types.NewLocalStorage("")
	fileNameWithoutExt := strings.Repeat("a", 100) + uuid.NewString()
	destinationLocation := storage.GetStorageLocation(
		filepath.Join(minioTestFolder, fileNameWithoutExt),
	)

	// When
	localStorageLocation, err := storage.PutObjectBytes(
		destinationLocation,
		bytes.NewBufferString(xmlContent),
	)

	// Then
	suite.Nil(err)
	defer localStorage.DeleteObject(localStorageLocation)
	defer storage.DeleteObject(destinationLocation)
}

func (suite *UnitTestSuite) TestMiniIOStorageGetObject() {
	// Given
	storage := types.NewMINIOStorage()
	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.GetStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorage.DeleteObject(localStorageLocation)
	suite.NotEmpty(localStorageLocation.FileName)
	suite.Contains(localStorageLocation.FileName, ".txt")

	remoteStorageLocation := storage.GetStorageLocation(
		filepath.Join(
			minioTestFolder,
			filepath.Base(localStorageLocation.FileName),
		),
	)
	localStorageLocation2, err := storage.PutObjectBytes(
		remoteStorageLocation,
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorage.DeleteObject(localStorageLocation2)
	defer storage.DeleteObject(remoteStorageLocation)

	receivedFileLocalStorageLocation := localStorage.GetStorageLocation(
		remoteStorageLocation.FileName,
	)

	// When
	err = storage.GetObject(
		storage.GetStorageLocation(
			filepath.Join(
				minioTestFolder,
				filepath.Base(localStorageLocation.FileName),
			),
		),
		receivedFileLocalStorageLocation,
	)

	// Then
	suite.Nil(err)
	defer localStorage.DeleteObject(receivedFileLocalStorageLocation)

	downloadedFileContent, err := os.ReadFile(
		filepath.Join(
			receivedFileLocalStorageLocation.LocalDirectory,
			receivedFileLocalStorageLocation.FileName,
		),
	)
	suite.Nil(err)
	suite.Equal(textContent, string(downloadedFileContent))
	suite.Contains(receivedFileLocalStorageLocation.FileName, ".txt")
}

func (suite *UnitTestSuite) TestMiniIOStorageGetObjectBytes() {
	// Given
	storage := types.NewMINIOStorage()
	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.GetStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorage.DeleteObject(localStorageLocation)
	suite.NotEmpty(localStorageLocation.FileName)
	suite.Contains(localStorageLocation.FileName, ".txt")

	remoteStorageLocation := storage.GetStorageLocation(
		filepath.Join(
			minioTestFolder,
			filepath.Base(localStorageLocation.FileName),
		),
	)
	localStorageLocation2, err := storage.PutObjectBytes(
		remoteStorageLocation,
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorage.DeleteObject(localStorageLocation2)
	defer storage.DeleteObject(remoteStorageLocation)

	// When
	downloadedFileContent, err := storage.GetObjectBytes(
		storage.GetStorageLocation(
			filepath.Join(
				minioTestFolder,
				filepath.Base(localStorageLocation.FileName),
			),
		),
	)

	// Then
	suite.Nil(err)
	suite.Equal(textContent, downloadedFileContent.String())
}

func (suite *UnitTestSuite) TestMiniIOStorageListObjects() {
	// Given
	storage := types.NewMINIOStorage()

	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.GetStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorage.DeleteObject(localStorageLocation)

	suite.NotEmpty(localStorageLocation.FileName)
	suite.Contains(localStorageLocation.FileName, ".txt")

	remoteStorageLocation := storage.GetStorageLocation(
		filepath.Join(
			minioTestFolder,
			localStorageLocation.FileName,
		),
	)
	err = storage.PutObject(
		localStorageLocation,
		remoteStorageLocation,
	)
	suite.Nil(err)
	defer storage.DeleteObject(remoteStorageLocation)

	// When
	bucketListing, err := storage.ListObjects(
		storage.GetStorageLocation(minioTestFolder),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(bucketListing)

	found := false
	for _, object := range bucketListing {
		if strings.Contains(object, localStorageLocation.FileName) {
			found = true
			break
		}
	}
	suite.True(found)
}
