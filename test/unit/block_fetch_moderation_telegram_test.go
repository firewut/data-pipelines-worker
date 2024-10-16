package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockFetchModerationFromTelegram() {
	block := blocks.NewBlockFetchModerationFromTelegram()

	suite.Equal("fetch_moderation_telegram", block.GetId())
	suite.Equal("Fetch Moderation from Telegram", block.GetName())
	suite.Equal("Fetch Moderation Action from Telegram", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal("", blockConfig.BlockSlug)
	suite.True(blockConfig.StopPipelineIfDecline)

	suite.True(blockConfig.RetryIfUnknown)
	suite.Equal(50, blockConfig.RetryCount)
	suite.Equal(time.Second*10, blockConfig.RetryInterval)
}

func (suite *UnitTestSuite) TestBlockFetchModerationFromTelegramValidateSchemaOk() {
	block := blocks.NewBlockFetchModerationFromTelegram()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockFetchModerationFromTelegramValidateSchemaFail() {
	block := blocks.NewBlockFetchModerationFromTelegram()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockFetchModerationFromTelegramProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockFetchModerationFromTelegram()
	data := &dataclasses.BlockData{
		Id:   "fetch_moderation_telegram",
		Slug: "fetch-moderation",
		Input: map[string]interface{}{
			"text": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorFetchModerationFromTelegram(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockFetchModerationFromTelegramProcessSuccess() {
	// Given
	processingId := uuid.New()
	processingInstanceId := uuid.New()

	block := blocks.NewBlockFetchModerationFromTelegram()
	blockConfig := block.GetBlockConfig(suite._config)

	data := &dataclasses.BlockData{
		Id:   "fetch_moderation_telegram",
		Slug: "fetch-moderation-decision",
		Input: map[string]interface{}{
			"block_slug":               "send-event-text-moderation-to-telegram",
			"stop_pipeline_if_decline": true,
		},
	}
	data.SetBlock(block)
	ctx := suite.GetContextWithcancel()
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, processingId)
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingInstanceID{}, processingInstanceId)

	cases := []struct {
		decisions []blocks.ModerationAction
		expected  blocks.ModerationAction
	}{
		{
			decisions: []blocks.ModerationAction{
				blocks.ShortenedActionApprove,
				blocks.ShortenedActionDecline,
			},
			expected: blocks.ModerationActionDecline,
		},
		{
			decisions: []blocks.ModerationAction{
				blocks.ShortenedActionApprove,
				blocks.ShortenedActionApprove,
			},
			expected: blocks.ModerationActionApprove,
		},
		{
			decisions: []blocks.ModerationAction{
				blocks.ShortenedActionApprove,
				blocks.ShortenedActionDecline,
				blocks.ShortenedActionApprove,
			},
			expected: blocks.ModerationActionApprove,
		},
		{
			decisions: []blocks.ModerationAction{
				blocks.ShortenedActionDecline,
				blocks.ShortenedActionApprove,
			},
			expected: blocks.ModerationActionApprove,
		},
		{
			decisions: []blocks.ModerationAction{
				blocks.ShortenedActionDecline,
				blocks.ShortenedActionDecline,
			},
			expected: blocks.ModerationActionDecline,
		},
		{
			decisions: []blocks.ModerationAction{
				blocks.ShortenedActionDecline,
				blocks.ShortenedActionApprove,
				blocks.ShortenedActionDecline,
			},
			expected: blocks.ModerationActionDecline,
		},
	}

	for indexCase, c := range cases {
		messages := make([]string, 0)
		for indexDecision, decision := range c.decisions {
			messages = append(
				messages,
				fmt.Sprintf(`
					{
						"callback_query": {
							"chat_instance": "111111111111111111",
							"data": "%s:%s:7470d33caf7ef9a794eba8cdf",
							"from": {
								"first_name": "John",
								"id": 987654321,
								"is_bot": false,
								"language_code": "en",
								"last_name": "Doe",
								"username": "johndoe"
							},
							"id": "%d",
							"message": {
								"chat": {
									"first_name": "John",
									"id": 987654321,
									"last_name": "Doe",
									"type": "private",
									"username": "johndoe"
								},
								"date": 1633044475,
								"message_id": 222,
								"text": "Please approve or reject"
							}
						},
						"update_id": 123456790
					}`,
					decision,
					processingId.String(),
					indexCase+indexDecision+1,
				),
			)
		}

		moderationDecisions := fmt.Sprintf(`
			{
				"ok": true,
				"result": [
					%s
				]
			}`,
			strings.Join(messages, ","),
		)

		telegramMockAPI := suite.GetMockHTTPServer(
			"",
			http.StatusOK,
			0,
			map[string]string{
				"/botTOKEN/getMe":      suite.GetTelegramBotInfo(),
				"/botTOKEN/getUpdates": moderationDecisions,
			},
		)
		telegramClient, err := factories.NewTelegramClient(telegramMockAPI.URL)
		suite.Nil(err)
		suite.NotNil(telegramClient)
		suite._config.Telegram.SetClient(telegramClient)

		// When
		result, stop, _, err := block.Process(
			ctx,
			blocks.NewProcessorFetchModerationFromTelegram(),
			data,
		)

		// Then
		suite.NotNil(result)

		if c.expected == blocks.ModerationActionApprove {
			suite.False(stop)
		} else {
			suite.Equal(blockConfig.StopPipelineIfDecline, stop)
		}
		suite.Nil(err)
		suite.Contains(result.String(), fmt.Sprintf(`"%s"`, c.expected))
	}
}

func (suite *UnitTestSuite) TestBlockFetchModerationFromTelegramProcessRetry() {
	// Given
	processingId := uuid.New()
	processingInstanceId := uuid.New()

	block := blocks.NewBlockFetchModerationFromTelegram()

	data := &dataclasses.BlockData{
		Id:   "fetch_moderation_telegram",
		Slug: "fetch-moderation-decision",
		Input: map[string]interface{}{
			"block_slug":               "send-event-text-moderation-to-telegram",
			"stop_pipeline_if_decline": true,
		},
	}
	data.SetBlock(block)
	ctx := suite.GetContextWithcancel()
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, processingId)
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingInstanceID{}, processingInstanceId)

	moderationDecisions := `{
		"ok": true,
		"result": [
			{
				"update_id": 123456789,
				"message": {
					"message_id": 111,
					"from": {
						"id": 987654321,
						"is_bot": false,
						"first_name": "John",
						"last_name": "Doe",
						"username": "johndoe",
						"language_code": "en"
					},
					"chat": {
						"id": 987654321,
						"first_name": "John",
						"last_name": "Doe",
						"username": "johndoe",
						"type": "private"
					},
					"date": 1633044474,
					"text": "This is a regular message"
				}
			}
		]
	}`

	telegramMockAPI := suite.GetMockHTTPServer(
		"",
		http.StatusOK,
		0,
		map[string]string{
			"/botTOKEN/getMe":      suite.GetTelegramBotInfo(),
			"/botTOKEN/getUpdates": moderationDecisions,
		},
	)
	telegramClient, err := factories.NewTelegramClient(telegramMockAPI.URL)
	suite.Nil(err)
	suite.NotNil(telegramClient)
	suite._config.Telegram.SetClient(telegramClient)

	// When
	result, stop, retry, err := block.Process(
		ctx,
		blocks.NewProcessorFetchModerationFromTelegram(),
		data,
	)

	// Then
	suite.True(retry)
	suite.NotNil(result)
	suite.False(stop)
	suite.Nil(err)
	suite.Contains(result.String(), fmt.Sprintf(`"%s"`, blocks.ModerationActionUnknown))
}
