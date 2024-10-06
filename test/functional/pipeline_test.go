package functional_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"

	"data-pipelines-worker/api/handlers"
	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

func (suite *FunctionalTestSuite) TestPipelineStartHandler() {
	// Given
	server, _, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)

	api_path := "/pipelines"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", api_path), nil)

	c := server.GetEcho().NewContext(req, rec)
	server.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(""))

	// When
	handlers.PipelinesHandler(server.GetPipelineRegistry())(c)

	// Then
	suite.Equal(http.StatusOK, rec.Code)
	suite.NotNil(rec.Body.String())
	suite.Contains(rec.Body.String(), "YT-CHANNEL-video-generation-block-prompt")
	suite.Contains(rec.Body.String(), "test-two-http-blocks")
}

func (suite *FunctionalTestSuite) TestPipelineStartHandlerTwoBlocks() {
	// Given
	server, _, err := suite.NewWorkerServerWithHandlers(true)
	suite.Nil(err)
	suite.NotEmpty(server)

	notificationChannel := make(chan interfaces.Processing)
	serverProcessingRegistry := server.GetProcessingRegistry()
	serverProcessingRegistry.SetNotificationChannel(notificationChannel)

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)
	server.GetPipelineRegistry().Add(suite.GetTestPipelineTwoBlocks(firstBlockInput))

	testPipelineSlug, _ := "test-two-http-blocks", "http_request"
	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: testPipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: "test-block-first-slug",
			Input: map[string]interface{}{
				"url": firstBlockInput,
			},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	// Wait for completion events
	block1Processing := <-notificationChannel
	suite.NotEmpty(block1Processing.GetId())
	suite.Equal(processingResponse.ProcessingID, block1Processing.GetId())

	block2Processing := <-notificationChannel
	suite.NotEmpty(block2Processing.GetId())
	suite.Equal(processingResponse.ProcessingID, block2Processing.GetId())
}

func (suite *FunctionalTestSuite) TestPipelineArrayFromJSONPathStart() {
	// Given
	pipelineSlug := "openai-test"

	openaiChatCompletionResponse := `{
		"id":"chatcmpl-123",
		"object":"chat.completion",
		"created":1677652288,
		"model":"gpt-4o-2024-08-06",
		"system_fingerprint":"fp_44709d6fcb",
		"choices":[
			{
				"index":0,
				"message":{
					"role":"assistant",
					"content":"On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry."
				},
				"logprobs":null,
				"finish_reason":"stop"
			}
		],
		"usage":{
			"prompt_tokens":9,
			"completion_tokens":12,
			"total_tokens":21,
			"completion_tokens_details":{
				"reasoning_tokens":0
			}
		}
	}`
	openaiTranscriptionResponse := `{"task":"transcribe","language":"english","duration":21.690000534057617,"segments":[{"id":0,"seek":0,"start":0,"end":8.140000343322754,"text":" On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.","tokens":[50364,1282,7617,1025,11,39498,11,264,1002,390,5680,3105,382,264,38376,4736,641,13828,2167,294,264,7051,13,50771],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":1,"seek":0,"start":8.140000343322754,"end":12.899999618530273,"text":" This marked the start of their legendary musical journey, leading to global fame.","tokens":[50771,639,12658,264,722,295,641,16698,9165,4671,11,5775,281,4338,16874,13,51009],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":2,"seek":0,"start":12.899999618530273,"end":16.739999771118164,"text":" Interestingly, John Lennon's harmonica playing added a distinct touch,","tokens":[51009,30564,11,2619,441,1857,266,311,14750,2262,2433,3869,257,10644,2557,11,51201],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":3,"seek":0,"start":16.739999771118164,"end":20.540000915527344,"text":" propelling them toward unprecedented stardom in the music industry.","tokens":[51201,25577,2669,552,7361,21555,342,515,298,294,264,1318,3518,13,51391],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false}],"words":null,"text":"On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry."}`
	openaiTTSRequestResponse := `tts-content`

	imageWidth, imageHeight := 256, 256
	pngBuffer := factories.GetPNGImageBuffer(imageWidth, imageHeight)
	openaiImageResponse := fmt.Sprintf(
		`{
			"created": 1683501845,
			"data": [
			  {
				"b64_json": "%s"
			  }
			]
		}`,
		base64.StdEncoding.EncodeToString(
			pngBuffer.Bytes(),
		),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(`{
			"data": [
				{"id": "gpt-3.5-turbo"},
				{"id": "text-davinci-003"},
				{"id": "text-curie-001"}
			]
		}`).Bytes())
	})

	mux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiChatCompletionResponse).Bytes())
	})

	mux.HandleFunc("/audio/speech", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "audio/mp3")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiTTSRequestResponse).Bytes())
	})

	mux.HandleFunc("/audio/transcriptions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiTranscriptionResponse).Bytes())
	})

	mux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiImageResponse).Bytes())
	})

	openAIMockServer := httptest.NewServer(mux)
	suite.httpTestServers = append(suite.httpTestServers, openAIMockServer)

	openaiMockClient := openai.NewClientWithConfig(
		openai.ClientConfig{
			BaseURL:            openAIMockServer.URL,
			APIType:            openai.APITypeOpenAI,
			AssistantVersion:   "v2",
			OrgID:              "",
			HTTPClient:         &http.Client{},
			EmptyMessagesLimit: 0,
		},
	)
	suite._config.OpenAI.SetClient(openaiMockClient)

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(
			`{
				"slug": "%s",
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
						"id": "openai_tts_request",
						"slug": "get-event-tts",
						"description": "Make a request to OpenAI TTS API to convert text to speech",
						"input_config": {
							"property": {
								"text": {
									"origin": "get-event-text",
									"jsonPath": "$"
								}
							}
						}
					},
					{
						"id": "openai_transcription_request",
						"slug": "get-event-transcription",
						"description": "Make a request to OpenAI TTS API to convert text to speech",
						"input_config": {
							"property": {
								"audio_file": {
									"origin": "get-event-tts"
								}
							}
						}
					},
					{
						"id": "openai_image_request",
						"slug": "get-event-image",
						"description": "Make a request to OpenAI Image API to get an image",
						"input_config": {
							"type": "array",
							"property": {
								"prompt": {
									"origin": "get-event-transcription",
									"jsonPath": "$.segments[*].text"
								}
							}
						},
						"input": {
							"quality": "hd",
							"size": "1024x1792"
						}
					}
				]
			}`,
			pipelineSlug,
		),
	)

	server, _, err := suite.NewWorkerServerWithHandlers(true, suite._config)
	suite.Nil(err)

	server.GetPipelineRegistry().Add(pipeline)
	notificationChannel := make(chan interfaces.Processing)
	processingRegistry := server.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: pipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug:  "get-event-text",
			Input: map[string]interface{}{},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	completedProcessing1 := <-notificationChannel
	suite.Equal(completedProcessing1.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing1.GetStatus())
	suite.Nil(completedProcessing1.GetError())

	completedProcessing2 := <-notificationChannel
	suite.Equal(completedProcessing2.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing2.GetStatus())
	suite.Nil(completedProcessing2.GetError())

	completedProcessing3 := <-notificationChannel
	suite.Equal(completedProcessing3.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing3.GetStatus())
	suite.Nil(completedProcessing3.GetError())

	for i := 0; i < 4; i++ {
		completedProcessing4 := <-notificationChannel
		suite.Equal(completedProcessing4.GetId(), processingResponse.ProcessingID)
		suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing4.GetStatus())
		suite.Nil(completedProcessing4.GetError())
	}
}

func (suite *FunctionalTestSuite) TestPipelineArrayFromJSONPathResume() {
	// Given
	pipelineSlug := "openai-test"
	processingId := uuid.New()

	openaiChatCompletionResponse := `{
		"id":"chatcmpl-123",
		"object":"chat.completion",
		"created":1677652288,
		"model":"gpt-4o-2024-08-06",
		"system_fingerprint":"fp_44709d6fcb",
		"choices":[
			{
				"index":0,
				"message":{
					"role":"assistant",
					"content":"On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry."
				},
				"logprobs":null,
				"finish_reason":"stop"
			}
		],
		"usage":{
			"prompt_tokens":9,
			"completion_tokens":12,
			"total_tokens":21,
			"completion_tokens_details":{
				"reasoning_tokens":0
			}
		}
	}`
	openaiTranscriptionResponse := `{"task":"transcribe","language":"english","duration":21.690000534057617,"segments":[{"id":0,"seek":0,"start":0,"end":8.140000343322754,"text":" On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.","tokens":[50364,1282,7617,1025,11,39498,11,264,1002,390,5680,3105,382,264,38376,4736,641,13828,2167,294,264,7051,13,50771],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":1,"seek":0,"start":8.140000343322754,"end":12.899999618530273,"text":" This marked the start of their legendary musical journey, leading to global fame.","tokens":[50771,639,12658,264,722,295,641,16698,9165,4671,11,5775,281,4338,16874,13,51009],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":2,"seek":0,"start":12.899999618530273,"end":16.739999771118164,"text":" Interestingly, John Lennon's harmonica playing added a distinct touch,","tokens":[51009,30564,11,2619,441,1857,266,311,14750,2262,2433,3869,257,10644,2557,11,51201],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":3,"seek":0,"start":16.739999771118164,"end":20.540000915527344,"text":" propelling them toward unprecedented stardom in the music industry.","tokens":[51201,25577,2669,552,7361,21555,342,515,298,294,264,1318,3518,13,51391],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false}],"words":null,"text":"On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry."}`
	openaiTTSRequestResponse := `tts-content`

	imageWidth, imageHeight := 256, 256
	pngBuffer := factories.GetPNGImageBuffer(imageWidth, imageHeight)
	openaiImageResponse := fmt.Sprintf(
		`{
			"created": 1683501845,
			"data": [
			  {
				"b64_json": "%s"
			  }
			]
		}`,
		base64.StdEncoding.EncodeToString(
			pngBuffer.Bytes(),
		),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(`{
			"data": [
				{"id": "gpt-3.5-turbo"},
				{"id": "text-davinci-003"},
				{"id": "text-curie-001"}
			]
		}`).Bytes())
	})

	mux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiImageResponse).Bytes())
	})

	openAIMockServer := httptest.NewServer(mux)
	suite.httpTestServers = append(suite.httpTestServers, openAIMockServer)

	openaiMockClient := openai.NewClientWithConfig(
		openai.ClientConfig{
			BaseURL:            openAIMockServer.URL,
			APIType:            openai.APITypeOpenAI,
			AssistantVersion:   "v2",
			OrgID:              "",
			HTTPClient:         &http.Client{},
			EmptyMessagesLimit: 0,
		},
	)
	suite._config.OpenAI.SetClient(openaiMockClient)

	pipeline := suite.GetTestPipeline(
		fmt.Sprintf(
			`{
				"slug": "%s",
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
						"id": "openai_tts_request",
						"slug": "get-event-tts",
						"description": "Make a request to OpenAI TTS API to convert text to speech",
						"input_config": {
							"property": {
								"text": {
									"origin": "get-event-text",
									"jsonPath": "$"
								}
							}
						}
					},
					{
						"id": "openai_transcription_request",
						"slug": "get-event-transcription",
						"description": "Make a request to OpenAI TTS API to convert text to speech",
						"input_config": {
							"property": {
								"audio_file": {
									"origin": "get-event-tts"
								}
							}
						}
					},
					{
						"id": "openai_image_request",
						"slug": "get-event-image",
						"description": "Make a request to OpenAI Image API to get an image",
						"input_config": {
							"type": "array",
							"property": {
								"prompt": {
									"origin": "get-event-transcription",
									"jsonPath": "$.segments[*].text"
								}
							}
						},
						"input": {
							"quality": "hd",
							"size": "1024x1792"
						}
					}
				]
			}`,
			pipelineSlug,
		),
	)

	server, _, err := suite.NewWorkerServerWithHandlers(true, suite._config)
	suite.Nil(err)

	server.GetPipelineRegistry().Add(pipeline)
	notificationChannel := make(chan interfaces.Processing)

	pipelineRegistry := server.GetPipelineRegistry()
	pipelineResultStorages := pipelineRegistry.GetPipelineResultStorages()
	processingRegistry := server.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)
	pipelineBlockDataRegistry := registries.NewPipelineBlockDataRegistry(
		processingId,
		pipelineSlug,
		pipelineResultStorages,
	)
	outputResults := map[string][]*bytes.Buffer{
		"get-event-text": {
			bytes.NewBufferString(openaiChatCompletionResponse),
		},
		"get-event-tts": {
			bytes.NewBufferString(openaiTTSRequestResponse),
		},
		"get-event-transcription": {
			bytes.NewBufferString(openaiTranscriptionResponse),
		},
	}
	for blockSlug, blockData := range outputResults {
		for _, data := range blockData {
			pipelineBlockDataRegistry.SaveOutput(blockSlug, 0, data)
			filePath := fmt.Sprintf(
				"%s/%s/%s",
				pipelineSlug,
				processingId,
				blockSlug,
			)

			for _, storage := range pipelineResultStorages {
				defer storage.DeleteObject(
					storage.NewStorageLocation(
						path.Join(
							filePath,
							fmt.Sprintf("output_%d", 0),
						),
					),
				)
			}
		}
	}

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug:         pipelineSlug,
			ProcessingID: processingId,
		},
		Block: schemas.BlockInputSchema{
			Slug:  "get-event-image",
			Input: map[string]interface{}{},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)
	suite.Equal(processingId, processingResponse.ProcessingID)

	for i := 0; i < 4; i++ {
		completedProcessing4 := <-notificationChannel
		suite.Equal(completedProcessing4.GetId(), processingResponse.ProcessingID)
		suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing4.GetStatus())
		suite.Nil(completedProcessing4.GetError())
	}
}
