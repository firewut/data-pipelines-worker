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

func (suite *UnitTestSuite) TestBlockSendMessageToTelegram() {
	block := blocks.NewBlockSendMessageToTelegram()

	suite.Equal("send_message_tg", block.GetId())
	suite.Equal("Send Message to Telegram", block.GetName())
	suite.Equal("Send Message to Telegram. In case of an image or video, the priority is given to the image.", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.NotNil(blockConfig.GroupId)
}

func (suite *UnitTestSuite) TestBlockSendMessageToTelegramValidateSchemaOk() {
	block := blocks.NewBlockSendMessageToTelegram()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockSendMessageToTelegramValidateSchemaFail() {
	block := blocks.NewBlockSendMessageToTelegram()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockSendMessageToTelegramProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockSendMessageToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_message_tg",
		Slug: "send-message",
		Input: map[string]interface{}{
			"text": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorSendMessageToTelegram(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockSendMessageToTelegramProcessSuccessTextDefaultDecisions() {
	// Given
	processingId := uuid.New()
	block := blocks.NewBlockSendMessageToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_message_tg",
		Slug: "send-message-text",
		Input: map[string]interface{}{
			"text":     "Hello world!",
			"group_id": 123456,
		},
	}
	data.SetBlock(block)
	ctx := suite.GetContextWithcancel()
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, processingId)

	// When
	result, stop, _, _, _, err := block.Process(
		ctx,
		blocks.NewProcessorSendMessageToTelegram(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Contains(result[0].String(), `"message_id"`)
}

func (suite *UnitTestSuite) TestBlockSendMessageToTelegramProcessSuccessTextWithImage() {
	// Given
	width := 100
	height := 100
	imageBuffer := factories.GetPNGImageBuffer(width, height)

	processingId := uuid.New()
	block := blocks.NewBlockSendMessageToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_message_tg",
		Slug: "send-message-text",
		Input: map[string]interface{}{
			"text":     "Hello world!",
			"group_id": 123456,
			"image":    imageBuffer.Bytes(),
		},
	}
	data.SetBlock(block)
	ctx := suite.GetContextWithcancel()
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, processingId)

	// When
	result, stop, _, _, _, err := block.Process(
		ctx,
		blocks.NewProcessorSendMessageToTelegram(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Contains(result[0].String(), `"message_id"`)
}

func (suite *UnitTestSuite) TestBlockSendMessageToTelegramProcessSuccessTextWithVideo() {
	// Given
	width := 100
	height := 100
	imageBuffer := factories.GetPNGImageBuffer(width, height)

	processingId := uuid.New()
	block := blocks.NewBlockSendMessageToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_message_tg",
		Slug: "send-message-text",
		Input: map[string]interface{}{
			"text":     "Hello world!",
			"group_id": 123456,
			"video":    imageBuffer.Bytes(),
		},
	}
	data.SetBlock(block)
	ctx := suite.GetContextWithcancel()
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, processingId)

	// When
	result, stop, _, _, _, err := block.Process(
		ctx,
		blocks.NewProcessorSendMessageToTelegram(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Contains(result[0].String(), `"message_id"`)
}

func (suite *UnitTestSuite) TestTelegramMessageGenerationBasic() {
	// Given
	review := blocks.TelegramMessage{
		Text:         "Please check the event.",
		ProcessingID: "30ec9fb3-8d63-43e2-84dc-9faf72a22253",
		BlockSlug:    "send-event-images-message-to-telegram",
		Index:        5,
	}

	// When
	generatedMessage := blocks.GenerateTelegramMessage(review)

	// Then
	suite.NotEmpty(generatedMessage)
	suite.Contains(generatedMessage, "Please check the event.")
	suite.Contains(generatedMessage, "ProcessingId: 30ec9fb3-8d63-43e2-84dc-9faf72a22253")
	suite.Contains(generatedMessage, "BlockSlug: send-event-images-message-to-telegram")
	suite.Contains(generatedMessage, "Index: 5")
}
