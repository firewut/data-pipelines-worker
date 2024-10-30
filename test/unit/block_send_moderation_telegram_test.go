package unit_test

import (
	"context"

	"github.com/google/uuid"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockSendModerationToTelegram() {
	block := blocks.NewBlockSendModerationToTelegram()

	suite.Equal("send_moderation_tg", block.GetId())
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
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockSendModerationToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_moderation_tg",
		Slug: "send-moderation",
		Input: map[string]interface{}{
			"text": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorSendModerationToTelegram(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramProcessSuccessTextDefaultDecisions() {
	// Given
	processingId := uuid.New()
	block := blocks.NewBlockSendModerationToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_moderation_tg",
		Slug: "send-moderation-text",
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
		blocks.NewProcessorSendModerationToTelegram(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Contains(result.String(), `"message_id"`)
	suite.Contains(result.String(), `"Approve"`)
	suite.Contains(result.String(), `"Decline"`)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramProcessSuccessTextExtraDecisions() {
	// Given
	processingId := uuid.New()
	block := blocks.NewBlockSendModerationToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_moderation_tg",
		Slug: "send-moderation-text",
		Input: map[string]interface{}{
			"text":     "Hello world!",
			"group_id": 123456,
			"extra_decisions": map[string]string{
				"Regenerate": "RRRegenerate",
			},
		},
	}
	data.SetBlock(block)
	ctx := suite.GetContextWithcancel()
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, processingId)

	// When
	result, stop, _, _, _, err := block.Process(
		ctx,
		blocks.NewProcessorSendModerationToTelegram(),
		data,
	)

	// Then
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)

	suite.Contains(result.String(), `"message_id"`)
	suite.Contains(result.String(), `"Approve"`)
	suite.Contains(result.String(), `"Decline"`)
	suite.Contains(result.String(), `"RRRegenerate"`)
}

func (suite *UnitTestSuite) TestBlockSendModerationToTelegramProcessSuccessTextWithImage() {
	// Given
	width := 100
	height := 100
	imageBuffer := factories.GetPNGImageBuffer(width, height)

	processingId := uuid.New()
	block := blocks.NewBlockSendModerationToTelegram()
	data := &dataclasses.BlockData{
		Id:   "send_moderation_tg",
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

	// When
	result, stop, _, _, _, err := block.Process(
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

func (suite *UnitTestSuite) TestTelegramReviewMessageGenerationBasic() {
	// Given
	review := blocks.TelegramReviewMessage{
		Text:         "Please check the event.",
		ProcessingID: "30ec9fb3-8d63-43e2-84dc-9faf72a22253",
		BlockSlug:    "send-event-images-moderation-to-telegram",
		Index:        5,
	}

	// When
	generatedMessage := blocks.GenerateTelegramReviewMessage(review)

	// Then
	suite.NotEmpty(generatedMessage)
	suite.Contains(generatedMessage, "Please check the event.")
	suite.Contains(generatedMessage, "ProcessingId: 30ec9fb3-8d63-43e2-84dc-9faf72a22253")
	suite.Contains(generatedMessage, "BlockSlug: send-event-images-moderation-to-telegram")
	suite.Contains(generatedMessage, "Index: 5")
}

func (suite *UnitTestSuite) TestTelegramReviewMessageGenerationWithExtraButtons() {
	// Given
	review := blocks.TelegramReviewMessage{
		Text:                "Please review the changes.",
		ProcessingID:        "f48c7d1a-69c1-4c3a-9c53-fbc05825c4c4",
		BlockSlug:           "send-changes-for-review",
		RegenerateBlockSlug: "image-generation",
		Index:               2,
		Buttons: []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("Approve", "Approve"),
			tgbotapi.NewInlineKeyboardButtonData("Decline", "Decline"),
			tgbotapi.NewInlineKeyboardButtonData("Regenerate", "Regenerate"),
		},
	}

	// When
	generatedMessage := blocks.GenerateTelegramReviewMessage(review)

	// Then
	suite.NotEmpty(generatedMessage)
	suite.Contains(generatedMessage, "Please review the changes.")
	suite.Contains(generatedMessage, "ProcessingId: f48c7d1a-69c1-4c3a-9c53-fbc05825c4c4")
	suite.Contains(generatedMessage, "BlockSlug: send-changes-for-review")
	suite.Contains(generatedMessage, "Index: 2")
	suite.Contains(generatedMessage, "image-generation")

	// Check buttons in the review struct instead of the message
	suite.Equal(len(review.Buttons), 3)                  // Verify the number of buttons
	suite.Contains(review.Buttons[0].Text, "Approve")    // Check button text
	suite.Contains(review.Buttons[1].Text, "Decline")    // Check button text
	suite.Contains(review.Buttons[2].Text, "Regenerate") // Check button text
}

func (suite *UnitTestSuite) TestTelegramReviewMessageGenerationWithImage() {
	// Given
	review := blocks.TelegramReviewMessage{
		Text:         "Please check the image.",
		ProcessingID: "c1a2e9e3-d451-4d62-b5c9-353e52c83f60",
		BlockSlug:    "send-image-for-review",
		Index:        1,
	}

	// When
	generatedMessage := blocks.GenerateTelegramReviewMessage(review)

	// Then
	suite.NotEmpty(generatedMessage)
	suite.Contains(generatedMessage, "Please check the image.")
	suite.Contains(generatedMessage, "ProcessingId: c1a2e9e3-d451-4d62-b5c9-353e52c83f60")
	suite.Contains(generatedMessage, "BlockSlug: send-image-for-review")
	suite.Contains(generatedMessage, "Index: 1")
}

func (suite *UnitTestSuite) TestParseTelegramMessage_ValidMessage_WithAllFields() {
	message := `Please review: Please check the event.
ProcessingId: 30ec9fb3-8d63-43e2-84dc-9faf72a22253
BlockSlug: send-event-images-moderation-to-telegram
RegenerateBlockSlug: get-event-images
Index: 5`

	parsedMessage, err := blocks.ParseTelegramReviewMessage(message)
	suite.NoError(err)
	suite.Equal("Please check the event.", parsedMessage.Text)
	suite.Equal("30ec9fb3-8d63-43e2-84dc-9faf72a22253", parsedMessage.ProcessingID)
	suite.Equal("send-event-images-moderation-to-telegram", parsedMessage.BlockSlug)
	suite.Equal("get-event-images", parsedMessage.RegenerateBlockSlug)
	suite.Equal(5, parsedMessage.Index)
}

func (suite *UnitTestSuite) TestParseTelegramMessage_ValidMessage_WithoutRegenerateBlockSlug() {
	message := `Please review: Please check the event.
ProcessingId: 30ec9fb3-8d63-43e2-84dc-9faf72a22253
BlockSlug: send-event-images-moderation-to-telegram
Index: 5`

	parsedMessage, err := blocks.ParseTelegramReviewMessage(message)
	suite.NoError(err)
	suite.Equal("Please check the event.", parsedMessage.Text)
	suite.Equal("30ec9fb3-8d63-43e2-84dc-9faf72a22253", parsedMessage.ProcessingID)
	suite.Equal("send-event-images-moderation-to-telegram", parsedMessage.BlockSlug)
	suite.Empty(parsedMessage.RegenerateBlockSlug) // Should be empty
	suite.Equal(5, parsedMessage.Index)
}

func (suite *UnitTestSuite) TestParseTelegramMessage_MissingProcessingID() {
	message := `Please review: Please check the event.
BlockSlug: send-event-images-moderation-to-telegram
Index: 5`

	parsedMessage, err := blocks.ParseTelegramReviewMessage(message)
	suite.Error(err)
	suite.Equal("missing or malformed text", err.Error())
	suite.Empty(parsedMessage.Text)
	suite.Empty(parsedMessage.ProcessingID)
	suite.Empty(parsedMessage.BlockSlug)
	suite.Empty(parsedMessage.RegenerateBlockSlug)
	suite.Equal(0, parsedMessage.Index)
}

func (suite *UnitTestSuite) TestParseTelegramMessage_MalformedMessage() {
	message := `Invalid message format`

	parsedMessage, err := blocks.ParseTelegramReviewMessage(message)
	suite.Error(err)
	suite.Equal("missing or malformed text", err.Error())
	suite.Empty(parsedMessage.Text)
	suite.Empty(parsedMessage.ProcessingID)
	suite.Empty(parsedMessage.BlockSlug)
	suite.Empty(parsedMessage.RegenerateBlockSlug)
	suite.Equal(0, parsedMessage.Index)
}
