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
	remoteTestBucket = "test-bucket"
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
	storage := types.NewLocalStorage("")

	// Test ListObjects
	objects, err := storage.ListObjects(os.TempDir())
	suite.NotNil(err)
	suite.Empty(objects)
}

func (suite *UnitTestSuite) TestLocalStoragePutObject() {
	storage := types.NewLocalStorage("")

	fileContent := bytes.NewBufferString(textContent)

	// Test PutObject
	err := storage.PutObject(os.TempDir(), fileContent.String(), "")
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestLocalStoragePutObjectBytes() {
	storage := types.NewLocalStorage("")

	fileContent := bytes.NewBufferString(textContent)

	// Test PutObjectBytes
	localFileName, err := storage.PutObjectBytes(os.TempDir(), fileContent, "")
	suite.Nil(err)
	suite.NotEmpty(localFileName)
	suite.Contains(localFileName, ".txt")
}

func (suite *UnitTestSuite) TestLocalStorageGetObjectBytes() {
	// Given
	storage := types.NewLocalStorage("")
	fileContent := bytes.NewBufferString(xmlContent)

	// When
	localFileName, err := storage.PutObjectBytes(os.TempDir(), fileContent, "")

	// Then
	suite.Nil(err)
	suite.NotEmpty(localFileName)
	suite.Contains(localFileName, ".xml")

	objectBytes, err := storage.GetObjectBytes(os.TempDir(), filepath.Base(localFileName))
	suite.Nil(err)
	suite.Equal(xmlContent, objectBytes.String())
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObject() {
	// Given
	storage := types.NewMINIOStorage()

	localFileName, err := types.NewLocalStorage("").PutObjectBytes(
		os.TempDir(),
		bytes.NewBufferString(textContent),
		"",
	)
	suite.Nil(err)
	suite.NotEmpty(localFileName)
	suite.Contains(localFileName, ".txt")

	putObjectAlias := fmt.Sprintf("%s/%s", remoteTestBucket, filepath.Base(localFileName))

	// When
	err = storage.PutObject(suite._config.Storage.Minio.Bucket, localFileName, putObjectAlias)

	// Then
	suite.Nil(err)

	localFilePath, err := storage.GetObject(suite._config.Storage.Minio.Bucket, putObjectAlias, localFileName)
	suite.Nil(err)
	defer os.Remove(localFilePath)

	downloadedFileContent, err := os.ReadFile(localFilePath)
	suite.Nil(err)
	suite.Equal(textContent, string(downloadedFileContent))
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObjectBytes() {
	// Given
	storage := types.NewMINIOStorage()

	fileNameNoExt := uuid.New().String()
	putObjectAlias := fmt.Sprintf("%s/%s", remoteTestBucket, fileNameNoExt)

	// When
	localFilePath, err := storage.PutObjectBytes(
		suite._config.Storage.Minio.Bucket,
		bytes.NewBufferString(xmlContent),
		putObjectAlias,
	)
	defer os.Remove(localFilePath)

	// Then
	suite.Nil(err)

	// Then
	bucketListing, err := storage.ListObjects(suite._config.Storage.Minio.Bucket)
	suite.Nil(err)
	suite.NotEmpty(bucketListing)

	var found string
	for _, object := range bucketListing {
		if strings.Contains(object, putObjectAlias) {
			found = object
			break
		}
	}
	suite.NotEmpty(found)
	suite.Contains(found, putObjectAlias)
	suite.Contains(found, ".xml")

	localFile, err := storage.GetObject(suite._config.Storage.Minio.Bucket, found, "")
	suite.Nil(err)
	defer os.Remove(localFile)
	suite.Contains(localFile, ".xml")

	downloadedFileContent, err := os.ReadFile(localFile)
	suite.Nil(err)
	suite.Equal(xmlContent, string(downloadedFileContent))
}

func (suite *UnitTestSuite) TestMiniIOStoragePutObjectBytesLongName() {
	// Given
	storage := types.NewMINIOStorage()

	fileNameNoExt := uuid.New().String()
	putObjectAlias := fmt.Sprintf("%s/%s", remoteTestBucket, fileNameNoExt)

	// When
	localFilePath, err := storage.PutObjectBytes(
		suite._config.Storage.Minio.Bucket,
		bytes.NewBufferString(xmlContent),
		putObjectAlias,
	)
	defer os.Remove(localFilePath)

	// Then
	suite.Nil(err)

	// Then
	bucketListing, err := storage.ListObjects("this-is-ignored")
	suite.Nil(err)
	suite.NotEmpty(bucketListing)

	var found string
	for _, object := range bucketListing {
		if strings.Contains(object, putObjectAlias) {
			found = object
			break
		}
	}
	suite.NotEmpty(found)
	suite.Contains(found, putObjectAlias)
	suite.Contains(found, ".xml")

	localFile, err := storage.GetObject(suite._config.Storage.Minio.Bucket, found, "")
	suite.Nil(err)
	defer os.Remove(localFile)
	suite.Contains(localFile, ".xml")

	downloadedFileContent, err := os.ReadFile(localFile)
	suite.Nil(err)
	suite.Equal(xmlContent, string(downloadedFileContent))
}

func (suite *UnitTestSuite) TestMiniIOStorageGetObject() {
	// Given
	storage := types.NewMINIOStorage()

	localFileName, err := types.NewLocalStorage("").PutObjectBytes(os.TempDir(), bytes.NewBufferString(textContent), "")
	suite.Nil(err)
	suite.NotEmpty(localFileName)
	suite.Contains(localFileName, ".txt")

	putObjectAlias := fmt.Sprintf("%s/%s", remoteTestBucket, filepath.Base(localFileName))
	localFilePath, err := storage.PutObjectBytes(
		suite._config.Storage.Minio.Bucket,
		bytes.NewBufferString(textContent),
		putObjectAlias,
	)
	defer os.Remove(localFilePath)
	suite.Nil(err)

	// When
	localFile, err := storage.GetObject(suite._config.Storage.Minio.Bucket, putObjectAlias, localFileName)

	// Then
	suite.Nil(err)
	defer os.Remove(localFile)

	downloadedFileContent, err := os.ReadFile(localFile)
	suite.Nil(err)
	suite.Equal(textContent, string(downloadedFileContent))
	suite.Contains(localFile, ".txt")
}

func (suite *UnitTestSuite) TestMiniIOStorageListObjects() {
	// Given
	storage := types.NewMINIOStorage()

	localFileName, err := types.NewLocalStorage("").PutObjectBytes(os.TempDir(), bytes.NewBufferString(textContent), "")
	suite.Nil(err)
	suite.NotEmpty(localFileName)
	suite.Contains(localFileName, ".txt")

	putObjectAlias := fmt.Sprintf("%s/%s", remoteTestBucket, filepath.Base(localFileName))
	localFilePath, err := storage.PutObjectBytes(
		suite._config.Storage.Minio.Bucket,
		bytes.NewBufferString(textContent),
		putObjectAlias,
	)
	defer os.Remove(localFilePath)
	suite.Nil(err)

	// When
	bucketListing, err := storage.ListObjects(suite._config.Storage.Minio.Bucket)

	// Then
	suite.Nil(err)
	suite.NotEmpty(bucketListing)

	found := false
	for _, object := range bucketListing {
		if object == putObjectAlias {
			found = true
			break
		}
	}
	suite.True(found)

}
