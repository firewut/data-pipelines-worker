package unit_test

import (
	"os"
	"path/filepath"

	"data-pipelines-worker/types"
)

func (suite *UnitTestSuite) TestS3Storage() {
	config := types.GetConfig()
	storage := types.NewMINIOStorage()

	tmpFile, err := os.CreateTemp("/tmp", "test")
	suite.Nil(err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString("test-source")
	suite.Nil(err)

	// Test PutObject
	err = storage.PutObject(config.Storage.Bucket, tmpFile.Name())
	suite.Nil(err)

	// Test ListObjects
	objects, err := storage.ListObjects(config.Storage.Bucket)
	suite.Nil(err)
	suite.Contains(objects, filepath.Base(tmpFile.Name()))

	// Test GetObject
	tmpFile2, err := os.CreateTemp("/tmp", "test-destination")
	suite.Nil(err)
	defer os.Remove(tmpFile2.Name())

	err = storage.GetObject(config.Storage.Bucket, filepath.Base(tmpFile.Name()), tmpFile2.Name())
	suite.Nil(err)

	// Check if the content is the same
	content1, err := os.ReadFile(tmpFile.Name())
	suite.Nil(err)
	content2, err := os.ReadFile(tmpFile2.Name())
	suite.Nil(err)
	suite.Equal(content1, content2)
}
