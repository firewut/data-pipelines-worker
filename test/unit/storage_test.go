package unit_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"data-pipelines-worker/types"
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
		mimeType, err := types.DetectMimeTypeFromBuffer(*buffer)
		suite.Nil(err)
		suite.Equal(_case.mimeType, mimeType.String())
		suite.Equal(_case.extension, mimeType.Extension())
	}
}

func (suite *UnitTestSuite) TestLocalStorageListObjectsNoDirectory() {
	// Given
	storage := types.NewLocalStorage("")

	tempDir, err := os.MkdirTemp(
		"",
		"test-nested-directory-*",
	)
	suite.Nil(err)
	suite.NotNil(tempDir)
	defer os.RemoveAll(tempDir)

	tempDir = filepath.Join(tempDir, "absent-directory")

	localLocation := storage.NewStorageLocation("")
	localLocation.SetLocalDirectory(tempDir)

	// When
	objects, err := storage.ListObjects(localLocation)

	// Then
	suite.NotNil(err)
	suite.Empty(objects)
}

func (suite *UnitTestSuite) TestLocalStorageListObjectsNoDirectoryDifferentLocalDirectory() {
	// Given
	storage := types.NewLocalStorage("")

	tempDir, err := os.MkdirTemp(
		"",
		"test-nested-directory-*",
	)
	suite.Nil(err)
	suite.NotNil(tempDir)
	defer os.RemoveAll(tempDir)

	tempDir = filepath.Join(tempDir, "absent-directory")

	localLocation := storage.NewStorageLocation("hewwo.txt")
	localLocation.SetLocalDirectory(tempDir)

	// When
	objects, err := storage.ListObjects(localLocation)

	// Then
	suite.NotNil(err)
	suite.Empty(objects)
}

func (suite *UnitTestSuite) TestLocalStorageListObjectsViaDirectory() {
	// Given
	storage := types.NewLocalStorage("")

	tempDir, err := os.MkdirTemp(
		"",
		"test-nested-directory-*",
	)
	suite.Nil(err)
	suite.NotNil(tempDir)
	defer os.RemoveAll(tempDir)

	tempFile, err := os.CreateTemp(tempDir, "tempfile-*.txt")
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return
	}
	defer tempFile.Close()

	localLocation := storage.NewStorageLocation("")
	localLocation.SetLocalDirectory(tempDir)

	// When
	objects, err := storage.ListObjects(localLocation)

	// Then
	suite.Nil(err)
	suite.NotEmpty(objects)
	for _, object := range objects {
		suite.Contains(object.GetFilePath(), filepath.Base(tempFile.Name()))

		info, err := os.Stat(object.GetFilePath())
		suite.Nil(err)
		suite.False(info.IsDir())
	}
}

func (suite *UnitTestSuite) TestLocalStorageListObjectsViaFileName() {
	// Given
	storage := types.NewLocalStorage("")

	tempDir, err := os.MkdirTemp(
		"",
		"test-nested-directory-*",
	)
	suite.Nil(err)
	suite.NotNil(tempDir)
	defer os.RemoveAll(tempDir)

	tempFile, err := os.CreateTemp(tempDir, "tempfile-*.txt")
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return
	}
	defer tempFile.Close()

	localLocation := storage.NewStorageLocation(tempFile.Name())
	localLocation.SetLocalDirectory("")

	// When
	objects, err := storage.ListObjects(localLocation)

	// Then
	suite.Nil(err)
	suite.NotEmpty(objects)
	for _, object := range objects {
		suite.Contains(object.GetFilePath(), filepath.Base(tempFile.Name()))

		info, err := os.Stat(object.GetFilePath())
		suite.Nil(err)
		suite.False(info.IsDir())
	}
}

func (suite *UnitTestSuite) TestLocalStoragePutObjectWithExt() {
	// Given
	fileContent := bytes.NewBufferString(textContent)
	oldFileName := fmt.Sprintf("%s.txt", uuid.NewString())
	newFileName := fmt.Sprintf("%s.txt", uuid.NewString())
	storage := types.NewLocalStorage("")

	file, err := os.Create(
		filepath.Join(
			storage.GetStorageDirectory(),
			oldFileName,
		),
	)
	suite.Nil(err)
	defer file.Close()
	_, err = file.Write(fileContent.Bytes())
	suite.Nil(err)

	source := storage.NewStorageLocation(oldFileName)
	destination := storage.NewStorageLocation(newFileName)

	// When
	destinationWithExtension, err := storage.PutObject(source, destination)

	// Then
	suite.Nil(err)
	defer destination.Delete()
	defer source.Delete()
	defer destinationWithExtension.Delete()

	// Ensure the new file exists and has the same content
	newFile, err := os.ReadFile(
		filepath.Join(
			storage.GetStorageDirectory(),
			newFileName,
		),
	)
	suite.Nil(err)
	suite.Equal(textContent, string(newFile))
}

func (suite *UnitTestSuite) TestLocalStoragePutObjectWithoutExt() {
	// Given
	fileContent := bytes.NewBufferString(textContent)
	oldFileName := uuid.NewString()
	newFileName := uuid.NewString()
	storage := types.NewLocalStorage("")

	file, err := os.Create(
		filepath.Join(
			storage.GetStorageDirectory(),
			oldFileName,
		),
	)
	suite.Nil(err)
	defer file.Close()
	_, err = file.Write(fileContent.Bytes())
	suite.Nil(err)

	source := storage.NewStorageLocation(oldFileName)
	destination := storage.NewStorageLocation(newFileName)

	// When
	destinationWithExtension, err := storage.PutObject(source, destination)

	// Then
	suite.Nil(err)
	suite.NotContains(destinationWithExtension.GetFileName(), ".txt")
	defer destination.Delete()
	defer source.Delete()
	defer destinationWithExtension.Delete()

	// Ensure the new file exists and has the same content
	newFile, err := os.ReadFile(
		filepath.Join(
			storage.GetStorageDirectory(),
			newFileName,
		),
	)
	suite.Nil(err)
	suite.Equal(textContent, string(newFile))
}

func (suite *UnitTestSuite) TestLocalStoragePutObjectBytes() {
	// Given
	storage := types.NewLocalStorage("")

	fileContent := bytes.NewBufferString(textContent)
	destinationLocation := storage.NewStorageLocation("")

	// When
	storageLocation, err := storage.PutObjectBytes(destinationLocation, fileContent)

	// Then
	suite.Nil(err)
	defer storageLocation.Delete()
	defer destinationLocation.Delete()

	suite.NotEmpty(storageLocation.GetFileName())
	suite.Contains(storageLocation.GetFileName(), ".txt")
}

func (suite *UnitTestSuite) TestLocalStorageGetObjectBytes() {
	// Given
	storage := types.NewLocalStorage("")
	fileContent := bytes.NewBufferString(xmlContent)
	destination := storage.NewStorageLocation("")

	// When
	storageLocation, err := storage.PutObjectBytes(destination, fileContent)

	// Then
	suite.Nil(err)
	defer storageLocation.Delete()
	defer destination.Delete()

	suite.NotEmpty(storageLocation.GetFileName())
	suite.Contains(storageLocation.GetFileName(), ".xml")

	objectBytes, err := storage.GetObjectBytes(storageLocation)
	suite.Nil(err)
	suite.Equal(xmlContent, objectBytes.String())
}

func (suite *UnitTestSuite) TestLocalStorageDeleteObject() {
	// Given
	storage := types.NewLocalStorage("")
	fileContent := bytes.NewBufferString(textContent)
	destination := storage.NewStorageLocation("")

	storageLocation, err := storage.PutObjectBytes(destination, fileContent)
	suite.Nil(err)
	suite.True(storageLocation.Exists())
	defer destination.Delete()

	// When
	err = storage.DeleteObject(storageLocation)

	// Then
	suite.Nil(err)
	suite.False(storageLocation.Exists())
}

func (suite *UnitTestSuite) TestLocalStorageShutDown() {
	// Given
	storage := types.NewLocalStorage("")

	// When
	storage.Shutdown()

	// Then
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObject() {
	suite.T().Skip("Skipping MiniIO due to Cache issues")

	// Given
	storage := types.NewMINIOStorage()

	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.NewStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorageLocation.Delete()

	suite.NotEmpty(localStorageLocation.GetFileName())
	suite.Contains(localStorageLocation.GetFileName(), ".txt")

	remoteStorageLocation := storage.NewStorageLocation(
		filepath.Join(
			minioTestFolder,
			localStorageLocation.GetFileName(),
		),
	)

	// When
	remoteStorageDestinationLocation, err := storage.PutObject(
		localStorageLocation,
		remoteStorageLocation,
	)

	// Then
	suite.Nil(err)
	defer remoteStorageLocation.Delete()
	defer remoteStorageDestinationLocation.Delete()
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObjectBytes() {
	suite.T().Skip("Skipping MiniIO due to Cache issues")

	// Given
	storage := types.NewMINIOStorage()

	fileNameWithoutExt := uuid.NewString()
	destinationLocation := storage.NewStorageLocation(
		filepath.Join(minioTestFolder, fileNameWithoutExt),
	)

	// When
	localStorageLocation, err := storage.PutObjectBytes(
		destinationLocation,
		bytes.NewBufferString(xmlContent),
	)

	// Then
	suite.Nil(err)
	defer destinationLocation.Delete()
	defer localStorageLocation.Delete()
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObjectBytesLongName() {
	suite.T().Skip("Skipping MiniIO due to Cache issues")

	// Given
	storage := types.NewMINIOStorage()

	fileNameWithoutExt := strings.Repeat("a", 100) + uuid.NewString()
	destinationLocation := storage.NewStorageLocation(
		filepath.Join(minioTestFolder, fileNameWithoutExt),
	)

	// When
	destinationWithExtension, err := storage.PutObjectBytes(
		destinationLocation,
		bytes.NewBufferString(xmlContent),
	)

	// Then
	suite.Nil(err)
	suite.Contains(destinationWithExtension.GetFileName(), ".xml")
	defer destinationLocation.Delete()
	defer destinationWithExtension.Delete()
}

func (suite *UnitTestSuite) TestMiniIOStorageGetObject() {
	suite.T().Skip("Skipping MiniIO due to Cache issues")

	// Given
	storage := types.NewMINIOStorage()
	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.NewStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorageLocation.Delete()
	suite.NotEmpty(localStorageLocation.GetFileName())
	suite.Contains(localStorageLocation.GetFileName(), ".txt")

	remoteStorageLocation := storage.NewStorageLocation(
		filepath.Join(
			minioTestFolder,
			filepath.Base(localStorageLocation.GetFileName()),
		),
	)
	localStorageLocation2, err := storage.PutObjectBytes(
		remoteStorageLocation,
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorageLocation2.Delete()
	defer remoteStorageLocation.Delete()

	receivedFileLocalStorageLocation := localStorage.NewStorageLocation(
		remoteStorageLocation.GetFileName(),
	)

	// When
	err = storage.GetObject(
		storage.NewStorageLocation(
			filepath.Join(
				minioTestFolder,
				filepath.Base(localStorageLocation.GetFileName()),
			),
		),
		receivedFileLocalStorageLocation,
	)

	// Then
	suite.Nil(err)
	defer receivedFileLocalStorageLocation.Delete()

	downloadedFileContent, err := os.ReadFile(
		filepath.Join(
			receivedFileLocalStorageLocation.GetLocalDirectory(),
			receivedFileLocalStorageLocation.GetFileName(),
		),
	)
	suite.Nil(err)
	suite.Equal(textContent, string(downloadedFileContent))
	suite.Contains(receivedFileLocalStorageLocation.GetFileName(), ".txt")
}

func (suite *UnitTestSuite) TestMiniIOStorageGetObjectBytes() {
	suite.T().Skip("Skipping MiniIO due to Cache issues")

	// Given
	storage := types.NewMINIOStorage()
	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.NewStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorageLocation.Delete()
	suite.NotEmpty(localStorageLocation.GetFileName())
	suite.Contains(localStorageLocation.GetFileName(), ".txt")

	remoteStorageLocation := storage.NewStorageLocation(
		filepath.Join(
			minioTestFolder,
			filepath.Base(localStorageLocation.GetFileName()),
		),
	)
	localStorageLocation2, err := storage.PutObjectBytes(
		remoteStorageLocation,
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorageLocation2.Delete()
	defer remoteStorageLocation.Delete()

	// When
	downloadedFileContent, err := storage.GetObjectBytes(
		storage.NewStorageLocation(
			filepath.Join(
				minioTestFolder,
				filepath.Base(localStorageLocation.GetFileName()),
			),
		),
	)

	// Then
	suite.Nil(err)
	suite.Equal(textContent, downloadedFileContent.String())
}

func (suite *UnitTestSuite) TestMiniIOStorageListObjects() {
	suite.T().Skip("Skipping MiniIO due to Cache issues")

	// Given
	storage := types.NewMINIOStorage()

	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.NewStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorageLocation.Delete()

	suite.NotEmpty(localStorageLocation.GetFileName())
	suite.Contains(localStorageLocation.GetFileName(), ".txt")

	remoteStorageLocation := storage.NewStorageLocation(
		filepath.Join(
			minioTestFolder,
			localStorageLocation.GetFileName(),
		),
	)
	_, err = storage.PutObject(
		localStorageLocation,
		remoteStorageLocation,
	)
	suite.Nil(err)
	defer remoteStorageLocation.Delete()

	// When
	bucketListing, err := storage.ListObjects(
		storage.NewStorageLocation(minioTestFolder),
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(bucketListing)

	found := false
	for _, object := range bucketListing {
		if strings.Contains(object.GetFileName(), localStorageLocation.GetFileName()) {
			found = true
			break
		}
	}
	suite.True(found)
}

func (suite *UnitTestSuite) TestMiniIOStorageDeleteObject() {
	suite.T().Skip("Skipping MiniIO due to Cache issues")

	// Given
	storage := types.NewMINIOStorage()

	localStorage := types.NewLocalStorage("")
	localStorageLocation, err := localStorage.PutObjectBytes(
		localStorage.NewStorageLocation(""),
		bytes.NewBufferString(textContent),
	)
	suite.Nil(err)
	defer localStorageLocation.Delete()

	suite.NotEmpty(localStorageLocation.GetFileName())
	suite.Contains(localStorageLocation.GetFileName(), ".txt")

	remoteStorageLocation := storage.NewStorageLocation(
		filepath.Join(
			minioTestFolder,
			localStorageLocation.GetFileName(),
		),
	)
	_, err = storage.PutObject(
		localStorageLocation,
		remoteStorageLocation,
	)
	suite.Nil(err)
	suite.True(remoteStorageLocation.Exists())

	// When
	err = storage.DeleteObject(remoteStorageLocation)

	// Then
	suite.Nil(err)
	suite.False(remoteStorageLocation.Exists())
}

func (suite *UnitTestSuite) TestMinioStorageShutDown() {
	// Given
	storage := types.NewMINIOStorage()

	// When
	storage.Shutdown()

	// Then
}
