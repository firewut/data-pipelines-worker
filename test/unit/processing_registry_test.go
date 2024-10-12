package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

func (suite *UnitTestSuite) TestNewProcessingRegistry() {
	registry := registries.NewProcessingRegistry()

	suite.NotNil(registry)
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestGetProcessingRegistry() {
	// Given
	cases := [][]bool{
		{false, false, true}, // Expect same instance
		{true, false, false}, // Expect the same instance as the one created
		{false, true, false}, // Expect different instances
		{true, true, false},  // Expect different instances
	}

	// When
	for _, c := range cases {
		processingRegistry1 := registries.GetProcessingRegistry(c[0])
		processingRegistry2 := registries.GetProcessingRegistry(c[1])

		// Then
		suite.Equal(processingRegistry1 == processingRegistry2, c[2])
	}
}

func (suite *UnitTestSuite) TestProcessingRegistryAdd() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)

	// When
	registry.Add(processing)

	// Then
	suite.NotEmpty(registry.GetAll())
	suite.Equal(1, len(registry.GetAll()))
	suite.Equal(processing, registry.Get(processingId.String()))
}

func (suite *UnitTestSuite) TestProcessingRegistryGetAll() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	registry.Add(processing)

	// When
	processings := registry.GetAll()

	// Then
	suite.NotEmpty(processings)
	suite.Equal(1, len(processings))
	suite.Equal(processing, processings[processing.GetId().String()])
}

func (suite *UnitTestSuite) TestProcessingRegistryGet() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	registry.Add(processing)

	// When
	result := registry.Get(processingId.String())

	// Then
	suite.NotNil(result)
	suite.Equal(processing, result)
}

func (suite *UnitTestSuite) TestProcessingRegistryDelete() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	// When
	registry.Delete(processingId.String())

	// Then
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestProcessingRegistryDeleteAll() {
	// Given
	processingId := uuid.New()
	registry := registries.NewProcessingRegistry()
	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		nil,
	)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	// When
	registry.DeleteAll()

	// Then
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestProcessingRegistryShutdownEmptyRegistry() {
	// Given
	registry := registries.NewProcessingRegistry()
	ctx := suite.GetShutDownContext(time.Second)

	// When
	err := registry.Shutdown(ctx)

	// Then
	suite.Nil(err)
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestProcessingRegistryStartProcessing() {
	// Given
	processingId := uuid.New()

	notificationChannel := make(chan interfaces.Processing)
	registry := registries.NewProcessingRegistry()
	registry.SetNotificationChannel(notificationChannel)

	block := blocks.NewBlockHTTP()

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, 0)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": successUrl,
		},
	)

	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	// registry.Add(processing)
	suite.Empty(registry.GetAll())

	go registry.StartProcessing(processing)

	// When
	completedProcessing := <-notificationChannel

	// Then
	suite.NotEmpty(registry.GetAll())
	suite.NotEmpty(completedProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing.GetStatus())
}

func (suite *UnitTestSuite) TestProcessingRegistryStartProcessingTwoBlocks() {
	// Given
	processingId := uuid.New()

	notificationChannel := make(chan interfaces.Processing)
	registry := registries.NewProcessingRegistry()
	registry.SetNotificationChannel(notificationChannel)

	block := blocks.NewBlockHTTP()
	blockId := block.GetId()

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, time.Millisecond)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineTwoBlocks(firstBlockInput),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": firstBlockInput,
		},
	)

	processing1 := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    blockId,
			Slug:  "test-block-first-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	processing2 := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    blockId,
			Slug:  "test-block-second-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing2.GetStatus())
	suite.Empty(registry.GetAll())

	// When
	go func() {
		processing1Output, stop, err := registry.StartProcessing(processing1)
		suite.Nil(err)
		suite.False(stop)
		suite.NotEmpty(processing1Output)
	}()
	completedProcessing1 := <-notificationChannel

	go func() {
		processing2Output, stop, err := registry.StartProcessing(processing2)
		suite.Nil(err)
		suite.False(stop)
		suite.NotEmpty(processing2Output)
	}()
	completedProcessing2 := <-notificationChannel

	// Then
	suite.NotEmpty(completedProcessing1.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing1.GetStatus())

	suite.NotEmpty(completedProcessing2.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing2.GetStatus())
}

func (suite *UnitTestSuite) TestProcessingRegistryShutdownCompletedProcessing() {
	// Given
	processingId := uuid.New()

	registry := registries.NewProcessingRegistry()

	block := blocks.NewBlockHTTP()

	processDelay := time.Millisecond * 10
	shutdownTriggerDelay := processDelay + time.Millisecond*5
	shutdownDelay := time.Millisecond * 20

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, processDelay)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": successUrl,
		},
	)
	ctx := suite.GetShutDownContext(shutdownDelay)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	go registry.StartProcessing(processing)

	time.Sleep(shutdownTriggerDelay)

	// When
	err := registry.Shutdown(ctx)

	// Then
	suite.Nil(err)
}

func (suite *UnitTestSuite) TestProcessingRegistryShutdownRunningProcessing() {
	// Given
	processingId := uuid.New()

	notificationChannel := make(chan interfaces.Processing)
	registry := registries.NewProcessingRegistry()
	registry.SetNotificationChannel(notificationChannel)

	block := blocks.NewBlockHTTP()

	processDelay := time.Hour
	shutdownTriggerDelay := time.Millisecond * 10
	shutdownDelay := time.Millisecond * 10

	successUrl := suite.GetMockHTTPServerURL("Hello, world!", http.StatusOK, processDelay)
	pipeline, inputDataSchema, _ := suite.RegisterTestPipelineAndInputForProcessing(
		suite.GetTestPipelineOneBlock(successUrl),
		"test-pipeline-slug",
		"test-block-slug",
		map[string]interface{}{
			"url": successUrl,
		},
	)
	ctx := suite.GetShutDownContext(shutdownDelay)
	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		&dataclasses.BlockData{
			Id:    block.GetId(),
			Slug:  "test-block-slug",
			Input: inputDataSchema.Block.Input,
		},
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	registry.Add(processing)
	suite.NotEmpty(registry.GetAll())

	go registry.StartProcessing(processing)

	time.Sleep(shutdownTriggerDelay)

	// When
	err := registry.Shutdown(ctx)

	// Then
	suite.NotNil(err)
	suite.Equal(context.DeadlineExceeded, err)
}

func (suite *UnitTestSuite) TestProcessingRegistryRetryProcessingFailed() {
	// Given
	processingId := uuid.New()
	retryCount := 3
	retryInterval := time.Millisecond

	notificationChannel := make(chan interfaces.Processing)
	registry := registries.NewProcessingRegistry()
	registry.SetNotificationChannel(notificationChannel)

	_config := config.GetConfig()
	_config.Blocks["fetch_moderation_from_telegram"].Config["retry_interval"] = retryInterval
	_config.Blocks["fetch_moderation_from_telegram"].Config["retry_count"] = retryCount

	block := blocks.NewBlockFetchModerationFromTelegram()
	pipeline := suite.GetTestPipeline(
		`{
			"slug": "openai-unit-test",
			"title": "Youtube video generation pipeline from prompt",
			"description": "Generates videos for youtube Channel <CHANNEL>. Uses Prompt in the Block.",
			"blocks": [
				{
					"id": "openai_chat_completion",
					"slug": "get-event-text",
					"description": "Get a text from OpenAI Chat Completion API",
					"input": {
						"model": "gpt-4o-2024-08-06",
						"system_prompt": "You must look for Historical event ( use google ) which happened today years ago. Write a short story about it. Add some interesting facts and make it engaging. The story MUST BE 15 words long!!!!!!!!",
						"user_prompt": "What happened years ago at date October 5 ?"
					}
				},
				{
					"id": "send_moderation_to_telegram",
					"slug": "send-event-text-moderation-to-telegram",
					"description": "Send the generated Event Text Content to Telegram for moderation",
					"input_config": {
						"property": {
							"text": {
								"origin": "get-event-text",
								"jsonPath": "$"
							}
						}
					},
					"input": {
						"group_id": -4573786981
					}
				},
				{
					"id": "fetch_moderation_from_telegram",
					"slug": "fetch-text-moderation-from-telegram",
					"description": "Fetch the moderation decision from Telegram",
					"input": {
						"block_slug": "send-event-text-moderation-to-telegram"
					}
				}
			]
		}`,
	)

	data := &dataclasses.BlockData{
		Id:   "fetch_moderation_from_telegram",
		Slug: "fetch-moderation-decision",
		Input: map[string]interface{}{
			"block_slug": "send-event-text-moderation-to-telegram",
		},
	}
	data.SetBlock(block)

	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		data,
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	suite.Empty(registry.GetAll())

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

	go registry.StartProcessing(processing)

	// When
	completedProcessing := <-notificationChannel

	// Then
	suite.NotEmpty(registry.GetAll())
	suite.NotEmpty(completedProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusRetryFailed, completedProcessing.GetStatus())
}

func (suite *UnitTestSuite) TestProcessingRegistryRetryProcessingSucceededFirstTry() {
	// Given
	processingId := uuid.New()
	retryCount := 3
	retryInterval := time.Millisecond

	notificationChannel := make(chan interfaces.Processing)
	registry := registries.NewProcessingRegistry()
	registry.SetNotificationChannel(notificationChannel)

	_config := config.GetConfig()
	_config.Blocks["fetch_moderation_from_telegram"].Config["retry_interval"] = retryInterval
	_config.Blocks["fetch_moderation_from_telegram"].Config["retry_count"] = retryCount

	block := blocks.NewBlockFetchModerationFromTelegram()
	pipeline := suite.GetTestPipeline(
		`{
			"slug": "openai-unit-test",
			"title": "Youtube video generation pipeline from prompt",
			"description": "Generates videos for youtube Channel <CHANNEL>. Uses Prompt in the Block.",
			"blocks": [
				{
					"id": "openai_chat_completion",
					"slug": "get-event-text",
					"description": "Get a text from OpenAI Chat Completion API",
					"input": {
						"model": "gpt-4o-2024-08-06",
						"system_prompt": "You must look for Historical event ( use google ) which happened today years ago. Write a short story about it. Add some interesting facts and make it engaging. The story MUST BE 15 words long!!!!!!!!",
						"user_prompt": "What happened years ago at date October 5 ?"
					}
				},
				{
					"id": "send_moderation_to_telegram",
					"slug": "send-event-text-moderation-to-telegram",
					"description": "Send the generated Event Text Content to Telegram for moderation",
					"input_config": {
						"property": {
							"text": {
								"origin": "get-event-text",
								"jsonPath": "$"
							}
						}
					},
					"input": {
						"group_id": -4573786981
					}
				},
				{
					"id": "fetch_moderation_from_telegram",
					"slug": "fetch-text-moderation-from-telegram",
					"description": "Fetch the moderation decision from Telegram",
					"input": {
						"block_slug": "send-event-text-moderation-to-telegram"
					}
				}
			]
		}`,
	)

	data := &dataclasses.BlockData{
		Id:   "fetch_moderation_from_telegram",
		Slug: "fetch-moderation-decision",
		Input: map[string]interface{}{
			"block_slug": "send-event-text-moderation-to-telegram",
		},
	}
	data.SetBlock(block)

	processing := dataclasses.NewProcessing(
		processingId,
		pipeline,
		block,
		data,
	)
	suite.Equal(interfaces.ProcessingStatusPending, processing.GetStatus())
	suite.Empty(registry.GetAll())

	moderationDecisions := fmt.Sprintf(`{
			"ok": true,
			"result": [
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
						"id": "123456789",
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
				}
			]
		}`,
		blocks.ShortenedActionApprove,
		processingId.String(),
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

	go registry.StartProcessing(processing)

	// When
	completedProcessing := <-notificationChannel

	// Then
	suite.NotEmpty(registry.GetAll())
	suite.NotEmpty(completedProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing.GetStatus())
}
