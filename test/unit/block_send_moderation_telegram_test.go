package unit_test

import (
	"context"

	"github.com/google/uuid"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockSendModerationToTelegram() {
	block := blocks.NewBlockSendModerationToTelegram()

	suite.Equal("send_moderation_telegram", block.GetId())
	suite.Equal("Send Moderation to Telegram", block.GetName())
	suite.Equal("Send Moderation Request to Telegram", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("Approve", blockConfig.Approve)
	suite.Equal("Decline", blockConfig.Decline)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramValidateSchemaOk() {
	block := blocks.NewBlockSendModerationToTelegram()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramValidateSchemaFail() {
	block := blocks.NewBlockSendModerationToTelegram()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockSendModerationToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_moderation_telegram",
		Slug: "send-moderation",
		Input: map[string]interface{}{
			"text": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorSendModerationToTelegram(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramProcessSuccessText() {
	// Given
	processingId := uuid.New()
	processingInstanceId := uuid.New()
	block := blocks.NewBlockSendModerationToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_moderation_telegram",
		Slug: "send-moderation-text",
		Input: map[string]interface{}{
			"text":     "Hello world!",
			"group_id": 123456,
		},
	}
	data.SetBlock(block)
	ctx := suite.GetContextWithcancel()
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, processingId)
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingInstanceID{}, processingInstanceId)

	// When
	result, stop, _, err := block.Process(
		ctx,
		blocks.NewProcessorSendModerationToTelegram(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Contains(result.String(), `"message_id"`)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramProcessSuccessTextWithImage() {
	// Given
	width := 100
	height := 100
	imageBuffer := factories.GetPNGImageBuffer(width, height)

	processingId := uuid.New()
	processingInstanceId := uuid.New()
	block := blocks.NewBlockSendModerationToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_moderation_telegram",
		Slug: "send-moderation-text",
		Input: map[string]interface{}{
			"text":     "Hello world!",
			"group_id": 123456,
			"image":    imageBuffer.Bytes(),
		},
	}
	data.SetBlock(block)
	ctx := suite.GetContextWithcancel()
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, processingId)
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingInstanceID{}, processingInstanceId)

	// When
	result, stop, _, err := block.Process(
		ctx,
		blocks.NewProcessorSendModerationToTelegram(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Contains(result.String(), `"message_id"`)
}
