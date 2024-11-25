package unit_test

import (
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockUploadFile() {
	block := blocks.NewBlockUploadFile()

	suite.Equal("upload_file", block.GetId())
	suite.Equal("Upload File", block.GetName())
	suite.Equal("Upload a file. This block is used to upload a file to the server and store it.", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())
}

func (suite *UnitTestSuite) TestBlockUploadFileValidateSchemaOk() {
	block := blocks.NewBlockUploadFile()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockUploadFileValidateSchemaFail() {
	block := blocks.NewBlockUploadFile()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockUploadFileProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockUploadFile()
	data := &dataclasses.BlockData{
		Id:   "upload_file",
		Slug: "upload-file",
		Input: map[string]interface{}{
			"file": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorUploadFile(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockUploadFileProcessSuccess() {
	// Given
	imageBuffer := factories.GetPNGImageBuffer(10, 10)

	block := blocks.NewBlockUploadFile()
	data := &dataclasses.BlockData{
		Id:   "upload_file",
		Slug: "upload-file",
		Input: map[string]interface{}{
			"file": imageBuffer.Bytes(),
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorUploadFile(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Equal(result, &imageBuffer)
}
