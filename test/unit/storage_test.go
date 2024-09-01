package unit_test

import (
	"bytes"
	"os"
	"path/filepath"

	"data-pipelines-worker/types"
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
	storage := types.NewLocalStorage()

	// Test ListObjects
	objects, err := storage.ListObjects(os.TempDir())
	suite.NotNil(err)
	suite.Empty(objects)
}

func (suite *UnitTestSuite) TestLocalStoragePutObject() {
	storage := types.NewLocalStorage()

	fileContent := bytes.NewBufferString("test-source")

	// Test PutObject
	err := storage.PutObject(
		os.TempDir(),
		fileContent.String(),
	)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestLocalStoragePutObjectBytes() {
	storage := types.NewLocalStorage()

	fileContent := bytes.NewBufferString("test-source")

	// Test PutObjectBytes
	localFileName, err := storage.PutObjectBytes(
		os.TempDir(),
		fileContent,
	)
	suite.Nil(err)
	suite.NotEmpty(localFileName)
	suite.Contains(localFileName, ".txt")
}

func (suite *UnitTestSuite) TestLocalStorageGetObjectBytes() {
	storage := types.NewLocalStorage()

	fileContent := bytes.NewBufferString("test-source")

	// Test PutObjectBytes
	localFileName, err := storage.PutObjectBytes(
		os.TempDir(),
		fileContent,
	)
	suite.Nil(err)
	suite.NotEmpty(localFileName)
	suite.Contains(localFileName, ".txt")

	objectBytes, err := storage.GetObjectBytes(
		os.TempDir(),
		filepath.Base(localFileName),
	)
	suite.Nil(err)
	suite.Equal(fileContent.String(), objectBytes.String())
}

func (suite *UnitTestSuite) TestMiniIOStorage() {
	storage := types.NewMINIOStorage()

	tmpFile, err := os.CreateTemp("/tmp", "test")
	suite.Nil(err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString("test-source")
	suite.Nil(err)

	// Test PutObject
	err = storage.PutObject(suite._config.Storage.Bucket, tmpFile.Name())
	suite.Nil(err)

	// Test PutObjectBytes
	localFileName, err := storage.PutObjectBytes(
		suite._config.Storage.Bucket,
		bytes.NewBufferString(xmlContent),
	)
	suite.Nil(err)
	suite.NotEmpty(localFileName)
	suite.Contains(localFileName, ".xml")

	// Test ListObjects
	objects, err := storage.ListObjects(suite._config.Storage.Bucket)
	suite.Nil(err)
	suite.Contains(objects, filepath.Base(tmpFile.Name()))

	// Test GetObject
	tmpFile2, err := os.CreateTemp("/tmp", "test-destination")
	suite.Nil(err)
	defer os.Remove(tmpFile2.Name())

	_, err = storage.GetObject(suite._config.Storage.Bucket, filepath.Base(tmpFile.Name()), tmpFile2.Name())
	suite.Nil(err)

	// Check if the content is the same
	content1, err := os.ReadFile(tmpFile.Name())
	suite.Nil(err)
	content2, err := os.ReadFile(tmpFile2.Name())
	suite.Nil(err)
	suite.Equal(content1, content2)
}
