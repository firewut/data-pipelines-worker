package functional_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"

	"data-pipelines-worker/api/handlers"
	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

func (suite *FunctionalTestSuite) TestPipelineStartHandler() {
	// Given
	server, _, err := suite.NewWorkerServerWithHandlers(true, suite._config)
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
	suite.Contains(rec.Body.String(), "openai-yt-short-generation")
	suite.Contains(rec.Body.String(), "test-two-http-blocks")
}

func (suite *FunctionalTestSuite) TestPipelineStartHandlerTwoBlocks() {
	// Given
	server, _, err := suite.NewWorkerServerWithHandlers(true, suite._config)
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
									"json_path": "$"
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
									"json_path": "$.segments[*].text"
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
		suite.Equal("openai_image_request", completedProcessing4.GetBlock().GetId())

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
									"json_path": "$"
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
									"json_path": "$.segments[*].text"
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
		for i, data := range blockData {
			pipelineBlockDataRegistry.SaveOutput(blockSlug, i, data)
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
							fmt.Sprintf("output_%d", i),
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
		suite.Equal("openai_image_request", completedProcessing4.GetBlock().GetId())

	}
}

func (suite *FunctionalTestSuite) TestPipelineArrayFromJSONPathReplaceStart() {
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

	prefix := "<PREFIX>"
	suffix := "<SUFFIX>"

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
									"json_path": "$"
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
						"id": "text_replace",
						"slug": "update-event-segments-for-image-prompt",
						"description": "Update the event segments for image prompt",
						"input_config": {
							"type": "array",
							"property": {
								"text": {
									"origin": "get-event-transcription",
									"json_path": "$.text"
								},
								"old": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								},
								"new": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"prefix": "%s",
							"suffix": "%s"
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
                        			"origin": "update-event-segments-for-image-prompt"
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
			prefix,
			suffix,
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

	updatedTexts := make([]string, 0)
	for i := 0; i < 4; i++ {
		completedProcessing4 := <-notificationChannel
		suite.Equal(completedProcessing4.GetId(), processingResponse.ProcessingID)
		suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing4.GetStatus())
		suite.Nil(completedProcessing4.GetError())
		suite.Equal("text_replace", completedProcessing4.GetBlock().GetId())
		output := completedProcessing4.GetOutput().GetValue().String()
		suite.Contains(output, prefix)
		suite.Contains(output, suffix)

		updatedTexts = append(updatedTexts, output)
	}

	for i := 0; i < 4; i++ {
		completedProcessing5 := <-notificationChannel
		suite.Equal(completedProcessing5.GetId(), processingResponse.ProcessingID)
		suite.Equal(interfaces.ProcessingStatusCompleted, completedProcessing5.GetStatus())
		suite.Nil(completedProcessing5.GetError())
		suite.Equal("openai_image_request", completedProcessing5.GetBlock().GetId())

		imageGenerationInputData := completedProcessing5.GetData().GetInputData().(map[string]interface{})
		suite.Equal(updatedTexts[i], imageGenerationInputData["prompt"])
	}
}

func (suite *FunctionalTestSuite) TestPipelineResumeArrayTargetIndex() {
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

	pngBuffer := factories.GetPNGImageBuffer(256, 256)
	img1buffer := factories.GetPNGImageBuffer(10, 10)
	img2buffer := factories.GetPNGImageBuffer(11, 11)
	img3buffer := factories.GetPNGImageBuffer(12, 12)
	img4buffer := factories.GetPNGImageBuffer(13, 13)
	finalImageBuffers := []*bytes.Buffer{
		&img1buffer,
		&img2buffer,
		&pngBuffer,
		&img4buffer,
	}

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
									"json_path": "$"
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
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"quality": "hd",
							"size": "1024x1792"
						}
					},
					{
						"id": "send_moderation_tg",
						"slug": "send-event-images-moderation-to-telegram",
						"description": "Send generated Event Images to Telegram for moderation",
						"input_config": {
							"type": "array",
							"property": {
								"image": {
									"origin": "get-event-image"
								},
								"text": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"group_id": -4573786981,
							"extra_decisions": {
								"Regenerate": "Regenerate It!"
							},
							"regenerate_block_slug": "get-event-image"
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
		"get-event-image": {
			&img1buffer,
			&img2buffer,
			&img3buffer,
			&img4buffer,
		},
	}
	for blockSlug, blockData := range outputResults {
		for i, data := range blockData {
			pipelineBlockDataRegistry.SaveOutput(blockSlug, i, data)
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
							fmt.Sprintf("output_%d", i),
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
			Slug:        "get-event-image",
			Input:       map[string]interface{}{},
			TargetIndex: 2,
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

	completedTargetProcessing := <-notificationChannel
	suite.Equal(completedTargetProcessing.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusCompleted, completedTargetProcessing.GetStatus())
	suite.Nil(completedTargetProcessing.GetError())
	suite.Equal("openai_image_request", completedTargetProcessing.GetBlock().GetId())
	suite.Equal(
		pngBuffer.Bytes(),
		completedTargetProcessing.GetOutput().GetValue().Bytes(),
	)

	// Ensure Registry data is distinct for block openai_image_request
	imagesBytes := pipelineBlockDataRegistry.LoadOutput("get-event-image")
	suite.Len(imagesBytes, 4)
	suite.Equal(imagesBytes, finalImageBuffers)

	for i := 0; i < 4; i++ {
		moderationProcessing := <-notificationChannel
		suite.Equal(moderationProcessing.GetId(), processingResponse.ProcessingID)
		suite.Equal(interfaces.ProcessingStatusCompleted, moderationProcessing.GetStatus())
		suite.Nil(moderationProcessing.GetError())
		suite.Equal("send_moderation_tg", moderationProcessing.GetBlock().GetId())

		blockData := moderationProcessing.GetData().GetInputData().(map[string]interface{})
		suite.Equal(finalImageBuffers[i].Bytes(), blockData["image"].([]byte))
	}

	select {
	case <-time.After(500 * time.Millisecond):
	case sideProcessing := <-notificationChannel:
		suite.Fail(
			fmt.Sprintf(
				"Expected notification channel to be empty, but got a value: %s",
				sideProcessing.GetOutput().GetValue().String(),
			),
		)
	}
}

func (suite *FunctionalTestSuite) TestPipelineArrayFromJSONPathStartParallel() {
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

	pngBuffer := factories.GetPNGImageBuffer(10, 10)
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

	imagesRequested := 0

	var mutex sync.Mutex
	mux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiImageResponse).Bytes())
		imagesRequested++
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
									"json_path": "$"
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
							"parallel": true,
							"property": {
								"prompt": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
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
	notificationChannel := make(chan interfaces.Processing, 100)
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
		suite.Equal("openai_image_request", completedProcessing4.GetBlock().GetId())
	}
	suite.Equal(4, imagesRequested)

	select {
	case <-time.After(500 * time.Millisecond):
	case sideProcessing := <-notificationChannel:
		suite.Fail(
			fmt.Sprintf(
				"Expected notification channel to be empty, but got a value: %s",
				sideProcessing.GetOutput().GetValue().String(),
			),
		)
	}
}

func (suite *FunctionalTestSuite) TestPipelineArrayFromJSONPathStartParallelFailAtIndex() {
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

	pngBuffer := factories.GetPNGImageBuffer(10, 10)
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

	imagesRequested := 0

	var mutex sync.Mutex
	mux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		// Fail responses after Second request
		if imagesRequested >= 2 {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiImageResponse).Bytes())
		imagesRequested++
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
									"json_path": "$"
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
							"parallel": true,
							"property": {
								"prompt": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
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
	notificationChannel := make(chan interfaces.Processing, 100)
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

	imageProcessings := make([]interfaces.Processing, 4)
	failedCount := 0
	succeedCount := 0
	for i := 0; i < 4; i++ {
		imageProcessing := <-notificationChannel
		imageProcessings[i] = imageProcessing
		suite.Equal("openai_image_request", imageProcessing.GetBlock().GetId())
		suite.Equal(imageProcessing.GetId(), processingResponse.ProcessingID)

		// Two of the image requests should fail
		if imageProcessing.GetStatus() == interfaces.ProcessingStatusFailed {
			failedCount++
			suite.NotNil(imageProcessing.GetError())
		} else if imageProcessing.GetStatus() == interfaces.ProcessingStatusCompleted {
			succeedCount++
			suite.Nil(imageProcessing.GetError())
		}
	}

	suite.Equal(2, failedCount)
	suite.Equal(2, succeedCount)

	select {
	case <-time.After(500 * time.Millisecond):
	case sideProcessing := <-notificationChannel:
		suite.Fail(
			fmt.Sprintf(
				"Expected notification channel to be empty, but got a value: %s",
				sideProcessing.GetOutput().GetValue().String(),
			),
		)
	}
}

func (suite *FunctionalTestSuite) TestPipelineArrayModerationApproveAll() {
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

	pngBuffer := factories.GetPNGImageBuffer(10, 10)
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

	openAIMux := http.NewServeMux()
	openAIMux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
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

	openAIMux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiChatCompletionResponse).Bytes())
	})

	openAIMux.HandleFunc("/audio/speech", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "audio/mp3")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiTTSRequestResponse).Bytes())
	})

	openAIMux.HandleFunc("/audio/transcriptions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiTranscriptionResponse).Bytes())
	})

	imagesRequested := 0

	var mutex sync.Mutex
	openAIMux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiImageResponse).Bytes())
		imagesRequested++
	})

	openAIMockServer := httptest.NewServer(openAIMux)
	suite.httpTestServers = append(suite.httpTestServers, openAIMockServer)

	openaiClient := factories.NewOpenAIClient(openAIMockServer.URL)
	suite._config.OpenAI.SetClient(openaiClient)

	// Telegram
	decisions := []string{
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionApprove,
	}
	moderationMessages := make([]string, 0)
	for i := 0; i < 4; i++ {
		moderationMatchCallbackData := fmt.Sprintf("%s:%d", decisions[i], i)
		reviewMessage := blocks.TelegramReviewMessage{
			Text: fmt.Sprintf(
				"Content for Review %d",
				i,
			),
			ProcessingID: processingId.String(),
			BlockSlug:    "send-event-images-moderation-to-telegram",
			Index:        i,
		}

		moderationMessages = append(
			moderationMessages,
			fmt.Sprintf(`
				{
					"callback_query": {
						"chat_instance": "%s",
						"data": "%s",
						"from": {
							"first_name": "John",
							"id": 987654321,
							"is_bot": false,
							"language_code": "en",
							"last_name": "Doe",
							"username": "johndoe"
						},
						"id": "%s",
						"message": {
							"chat": {
								"first_name": "John",
								"id": 987654321,
								"last_name": "Doe",
								"type": "private",
								"username": "johndoe"
							},
							"date": 1633044475,
							"message_id": %d,
							"text": "%s"
						}
					},
					"update_id": 123456790
				}`,
				uuid.New().String(),
				moderationMatchCallbackData,
				uuid.New().String(),
				i,
				blocks.FormatTelegramMessage(
					blocks.GenerateTelegramReviewMessage(reviewMessage),
				),
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
		strings.Join(moderationMessages, ","),
	)

	telegramMockAPI := suite.GetMockHTTPServer(
		"",
		http.StatusOK,
		0,
		map[string][]string{
			"/botTOKEN/getMe":      {suite.GetTelegramBotInfo()},
			"/botTOKEN/getUpdates": {moderationDecisions},
			"/botTOKEN/sendMessage": {`{
				"ok": true,
				"result": {
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
			}`},
		},
	)
	telegramClient, err := factories.NewTelegramClient(telegramMockAPI.URL)
	suite.Nil(err)
	suite.NotNil(telegramClient)
	suite._config.Telegram.SetClient(telegramClient)

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
									"json_path": "$"
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
						"slug": "get-event-images",
						"description": "Make a request to OpenAI Image API to get an image",
						"input_config": {
							"type": "array",
							"parallel": true,
							"property": {
								"prompt": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"quality": "hd",
							"size": "1024x1792"
						}
					},
					{
						"id": "send_moderation_tg",
						"slug": "send-event-images-moderation-to-telegram",
						"description": "Send generated Event Images to Telegram for moderation",
						"input_config": {
							"type": "array",
							"parallel": true,
							"property": {
								"image": {
									"origin": "get-event-images"
								},
								"text": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"group_id": -4573786981,
							"extra_decisions": {
								"Regenerate": "Refresh"
							}
						}
					},
					{
						"id": "fetch_moderation_tg",
						"slug": "fetch-event-images-moderation-from-telegram",
						"description": "Fetch the moderation Image decision from Telegram",
						"input_config": {
							"type": "array",
							"property": {
								"hack_for_array_trigger": {
									"origin": "send-event-images-moderation-to-telegram"
								}
							}
						},
						"input": {
							"block_slug": "send-event-images-moderation-to-telegram"
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
	notificationChannel := make(chan interfaces.Processing, 100)
	processingRegistry := server.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug:         pipelineSlug,
			ProcessingID: processingId,
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
	suite.Equal(processingId, processingResponse.ProcessingID)

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
		imageProcessing := <-notificationChannel
		suite.Equal("openai_image_request", imageProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, imageProcessing.GetId())

		suite.Equal(interfaces.ProcessingStatusCompleted, imageProcessing.GetStatus())
		suite.Nil(imageProcessing.GetError())
	}

	for i := 0; i < 4; i++ {
		sendModerationProcessing := <-notificationChannel
		suite.Equal("send_moderation_tg", sendModerationProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, sendModerationProcessing.GetId())

		suite.Equal(interfaces.ProcessingStatusCompleted, sendModerationProcessing.GetStatus())
		suite.Nil(sendModerationProcessing.GetError())
	}

	for i := 0; i < 4; i++ {
		fetchModerationProcessing := <-notificationChannel
		suite.Equal("fetch_moderation_tg", fetchModerationProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, fetchModerationProcessing.GetId())

		suite.Equal(interfaces.ProcessingStatusCompleted, fetchModerationProcessing.GetStatus())
		suite.Nil(fetchModerationProcessing.GetError())
	}

	select {
	case <-time.After(500 * time.Millisecond):
	case sideProcessing := <-notificationChannel:
		suite.Fail(
			fmt.Sprintf(
				"Expected notification channel to be empty, but got a value: %s",
				sideProcessing.GetOutput().GetValue().String(),
			),
		)
	}
}

func (suite *FunctionalTestSuite) TestPipelineArrayModerationDeclineThird() {
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

	pngBuffer := factories.GetPNGImageBuffer(10, 10)
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

	openAIMux := http.NewServeMux()
	openAIMux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
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

	openAIMux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiChatCompletionResponse).Bytes())
	})

	openAIMux.HandleFunc("/audio/speech", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "audio/mp3")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiTTSRequestResponse).Bytes())
	})

	openAIMux.HandleFunc("/audio/transcriptions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiTranscriptionResponse).Bytes())
	})

	imagesRequested := 0

	var mutex sync.Mutex
	openAIMux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiImageResponse).Bytes())
		imagesRequested++
	})

	openAIMockServer := httptest.NewServer(openAIMux)
	suite.httpTestServers = append(suite.httpTestServers, openAIMockServer)

	openaiClient := factories.NewOpenAIClient(openAIMockServer.URL)
	suite._config.OpenAI.SetClient(openaiClient)

	// Telegram
	decisions := []string{
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionDecline,
		blocks.ShortenedActionApprove,
	}
	moderationMessages := make([]string, 0)
	for i := 0; i < 4; i++ {
		moderationMatchCallbackData := fmt.Sprintf("%s:%d", decisions[i], i)
		reviewMessage := blocks.TelegramReviewMessage{
			Text: fmt.Sprintf(
				"Content for Review %d",
				i,
			),
			ProcessingID: processingId.String(),
			BlockSlug:    "send-event-images-moderation-to-telegram",
			Index:        i,
		}

		moderationMessages = append(
			moderationMessages,
			fmt.Sprintf(`
				{
					"callback_query": {
						"chat_instance": "111111111111111111",
						"data": "%s",
						"from": {
							"first_name": "John",
							"id": 987654321,
							"is_bot": false,
							"language_code": "en",
							"last_name": "Doe",
							"username": "johndoe"
						},
						"id": "%s-%d",
						"message": {
							"chat": {
								"first_name": "John",
								"id": 987654321,
								"last_name": "Doe",
								"type": "private",
								"username": "johndoe"
							},
							"date": 1633044475,
							"message_id": %d,
							"text": "%s"
						}
					},
					"update_id": 123456790
				}`,
				moderationMatchCallbackData,
				uuid.New().String(),
				i,
				i,
				blocks.FormatTelegramMessage(
					blocks.GenerateTelegramReviewMessage(reviewMessage),
				),
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
		strings.Join(moderationMessages, ","),
	)

	telegramMockAPI := suite.GetMockHTTPServer(
		"",
		http.StatusOK,
		0,
		map[string][]string{
			"/botTOKEN/getMe":      {suite.GetTelegramBotInfo()},
			"/botTOKEN/getUpdates": {moderationDecisions},
			"/botTOKEN/sendMessage": {`{
				"ok": true,
				"result": {
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
			
			}`},
		},
	)
	telegramClient, err := factories.NewTelegramClient(telegramMockAPI.URL)
	suite.Nil(err)
	suite.NotNil(telegramClient)
	suite._config.Telegram.SetClient(telegramClient)

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
									"json_path": "$"
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
						"slug": "get-event-images",
						"description": "Make a request to OpenAI Image API to get an image",
						"input_config": {
							"type": "array",
							"parallel": true,
							"property": {
								"prompt": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"quality": "hd",
							"size": "1024x1792"
						}
					},
					{
						"id": "send_moderation_tg",
						"slug": "send-event-images-moderation-to-telegram",
						"description": "Send generated Event Images to Telegram for moderation",
						"input_config": {
							"type": "array",
							"parallel": true,
							"property": {
								"image": {
									"origin": "get-event-images"
								},
								"text": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"group_id": -4573786981,
							"extra_decisions": {
								"Regenerate": "Refresh"
							}
						}
					},
					{
						"id": "fetch_moderation_tg",
						"slug": "fetch-event-images-moderation-from-telegram",
						"description": "Fetch the moderation Image decision from Telegram",
						"input_config": {
							"type": "array",
							"property": {
								"hack_for_array_trigger": {
									"origin": "send-event-images-moderation-to-telegram"
								}
							}
						},
						"input": {
							"block_slug": "send-event-images-moderation-to-telegram"
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
	notificationChannel := make(chan interfaces.Processing, 100)
	processingRegistry := server.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug:         pipelineSlug,
			ProcessingID: processingId,
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
	suite.Equal(processingId, processingResponse.ProcessingID)

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
		imageProcessing := <-notificationChannel
		suite.Equal("openai_image_request", imageProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, imageProcessing.GetId())

		suite.Equal(interfaces.ProcessingStatusCompleted, imageProcessing.GetStatus())
		suite.Nil(imageProcessing.GetError())
	}

	for i := 0; i < 4; i++ {
		sendModerationProcessing := <-notificationChannel
		suite.Equal("send_moderation_tg", sendModerationProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, sendModerationProcessing.GetId())

		suite.Equal(interfaces.ProcessingStatusCompleted, sendModerationProcessing.GetStatus())
		suite.Nil(sendModerationProcessing.GetError())
	}

	for i := 0; i < 3; i++ {
		fetchModerationProcessing := <-notificationChannel
		suite.Equal("fetch_moderation_tg", fetchModerationProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, fetchModerationProcessing.GetId())

		if i == 2 {
			suite.Equal(interfaces.ProcessingStatusStopped, fetchModerationProcessing.GetStatus(), i)
		} else {
			suite.Equal(interfaces.ProcessingStatusCompleted, fetchModerationProcessing.GetStatus(), i)
		}
		suite.Nil(fetchModerationProcessing.GetError())
	}

	select {
	case <-time.After(500 * time.Millisecond):
	case sideProcessing := <-notificationChannel:
		suite.Fail(
			fmt.Sprintf(
				"Expected notification channel to be empty, but got a value: %s",
				sideProcessing.GetOutput().GetValue().String(),
			),
		)
	}
}

func (suite *FunctionalTestSuite) TestPipelineArrayModerationRegenerateThird() {
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

	pngBuffer := factories.GetPNGImageBuffer(10, 10)
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

	openAIMux := http.NewServeMux()
	openAIMux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
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

	openAIMux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiChatCompletionResponse).Bytes())
	})

	openAIMux.HandleFunc("/audio/speech", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "audio/mp3")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiTTSRequestResponse).Bytes())
	})

	openAIMux.HandleFunc("/audio/transcriptions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiTranscriptionResponse).Bytes())
	})

	imagesRequested := 0

	var mutex sync.Mutex
	openAIMux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes.NewBufferString(openaiImageResponse).Bytes())
		imagesRequested++
	})

	openAIMockServer := httptest.NewServer(openAIMux)
	suite.httpTestServers = append(suite.httpTestServers, openAIMockServer)

	openaiClient := factories.NewOpenAIClient(openAIMockServer.URL)
	suite._config.OpenAI.SetClient(openaiClient)

	// Moderation Decisions
	decisionsRegenerate := []string{
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionRegenerate,
		blocks.ShortenedActionApprove,
	}
	decisionsApprove := []string{
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionApprove,
		blocks.ShortenedActionApprove,
	}
	telegramResponseTemplate := `{
		"ok": true,
		"result": [
			%s
		]
	}`
	messageTemplate := `{
		"callback_query": {
			"chat_instance": "111111111111111111",
			"data": "%s",
			"from": {
				"first_name": "John",
				"id": 987654321,
				"is_bot": false,
				"language_code": "en",
				"last_name": "Doe",
				"username": "johndoe"
			},
			"id": "%s",
			"message": {
				"chat": {
					"first_name": "John",
					"id": 987654321,
					"last_name": "Doe",
					"type": "private",
					"username": "johndoe"
				},
				"date": 1633044475,
				"message_id": %d,
				"text": "%s"
			}
		},
		"update_id": 123456790
	}`
	// Telegram
	moderationRegenerateMessages := make([]string, 0)
	moderationApproveMessages := make([]string, 0)
	for i := 0; i < 4; i++ {
		reviewMessage := blocks.TelegramReviewMessage{
			Text: fmt.Sprintf(
				"Content for Review %d",
				i,
			),
			ProcessingID:        processingId.String(),
			BlockSlug:           "send-event-images-moderation-to-telegram",
			RegenerateBlockSlug: "get-event-images",
			Index:               i,
		}

		messageWithRegenerate := fmt.Sprintf(messageTemplate,
			fmt.Sprintf("%s:%d", decisionsRegenerate[i], i),
			uuid.New().String(),
			i,
			blocks.FormatTelegramMessage(
				blocks.GenerateTelegramReviewMessage(reviewMessage),
			),
		)

		messageWithApprove := fmt.Sprintf(messageTemplate,
			fmt.Sprintf("%s:%d", decisionsApprove[i], i),
			uuid.New().String(),
			i+10,
			blocks.FormatTelegramMessage(
				blocks.GenerateTelegramReviewMessage(reviewMessage),
			),
		)

		moderationApproveMessages = append(moderationApproveMessages, messageWithApprove)
		moderationRegenerateMessages = append(moderationRegenerateMessages, messageWithRegenerate)
	}

	telegramMockAPI := suite.GetMockHTTPServer(
		"",
		http.StatusOK,
		0,
		map[string][]string{
			"/botTOKEN/getMe": {
				suite.GetTelegramBotInfo(),
			},
			"/botTOKEN/getUpdates": {
				fmt.Sprintf(telegramResponseTemplate, moderationRegenerateMessages[0]),
				fmt.Sprintf(telegramResponseTemplate, moderationRegenerateMessages[1]),
				fmt.Sprintf(telegramResponseTemplate, moderationRegenerateMessages[2]),
				fmt.Sprintf(telegramResponseTemplate, moderationApproveMessages[0]),
				fmt.Sprintf(telegramResponseTemplate, moderationApproveMessages[1]),
				fmt.Sprintf(telegramResponseTemplate, moderationApproveMessages[2]),
				fmt.Sprintf(telegramResponseTemplate, moderationApproveMessages[3]),
			},
			"/botTOKEN/sendMessage": {`{
				"ok": true,
				"result": {
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
			
			}`},
		},
	)
	telegramClient, err := factories.NewTelegramClient(telegramMockAPI.URL)
	suite.Nil(err)
	suite.NotNil(telegramClient)
	suite._config.Telegram.SetClient(telegramClient)

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
									"json_path": "$"
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
						"slug": "get-event-images",
						"description": "Make a request to OpenAI Image API to get an image",
						"input_config": {
							"type": "array",
							"property": {
								"prompt": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"quality": "hd",
							"size": "1024x1792"
						}
					},
					{
						"id": "send_moderation_tg",
						"slug": "send-event-images-moderation-to-telegram",
						"description": "Send generated Event Images to Telegram for moderation",
						"input_config": {
							"type": "array",
							"property": {
								"image": {
									"origin": "get-event-images"
								},
								"text": {
									"origin": "get-event-transcription",
									"json_path": "$.segments[*].text"
								}
							}
						},
						"input": {
							"group_id": -4573786981,
							"extra_decisions": {
								"Regenerate": "Refresh"
							},
							"regenerate_block_slug": "get-event-images"
						}
					},
					{
						"id": "fetch_moderation_tg",
						"slug": "fetch-event-images-moderation-from-telegram",
						"description": "Fetch the moderation Image decision from Telegram",
						"input_config": {
							"type": "array",
							"property": {
								"hack_for_array_trigger": {
									"origin": "send-event-images-moderation-to-telegram"
								}
							}
						},
						"input": {
							"block_slug": "send-event-images-moderation-to-telegram"
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
	notificationChannel := make(chan interfaces.Processing, 100)
	processingRegistry := server.GetProcessingRegistry()
	processingRegistry.SetNotificationChannel(notificationChannel)

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug:         pipelineSlug,
			ProcessingID: processingId,
		},
		Block: schemas.BlockInputSchema{
			Slug: "get-event-text",
			Input: map[string]interface{}{
				"user_prompt": "get-event-text-input",
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
	suite.Equal(processingId, processingResponse.ProcessingID)

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
		imageProcessing := <-notificationChannel
		suite.Equal("openai_image_request", imageProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, imageProcessing.GetId())

		suite.Equal(interfaces.ProcessingStatusCompleted, imageProcessing.GetStatus())
		suite.Nil(imageProcessing.GetError())
	}
	suite.Equal(4, imagesRequested)

	for i := 0; i < 4; i++ {
		sendModerationProcessing := <-notificationChannel
		suite.Equal("send_moderation_tg", sendModerationProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, sendModerationProcessing.GetId())

		suite.Equal(interfaces.ProcessingStatusCompleted, sendModerationProcessing.GetStatus())
		suite.Nil(sendModerationProcessing.GetError())
	}

	moderationStatuses := []interfaces.ProcessingStatus{
		interfaces.ProcessingStatusCompleted,
		interfaces.ProcessingStatusCompleted,
		interfaces.ProcessingStatusStoppedForRegeneration,
	}
	for i := 0; i < 3; i++ {
		fetchModerationProcessing := <-notificationChannel
		suite.Equal("fetch_moderation_tg", fetchModerationProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, fetchModerationProcessing.GetId())

		suite.Equal(moderationStatuses[i], fetchModerationProcessing.GetStatus(), fetchModerationProcessing.GetInstanceId().String())
		suite.Nil(fetchModerationProcessing.GetError())
	}

	imageProcessing := <-notificationChannel
	suite.Equal("openai_image_request", imageProcessing.GetBlock().GetId())
	suite.Equal(processingResponse.ProcessingID, imageProcessing.GetId())
	suite.Equal(interfaces.ProcessingStatusCompleted, imageProcessing.GetStatus())
	suite.Nil(imageProcessing.GetError())
	suite.Equal(5, imagesRequested)

	for i := 0; i < 4; i++ {
		sendModerationProcessing := <-notificationChannel
		suite.Equal("send_moderation_tg", sendModerationProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, sendModerationProcessing.GetId())
		suite.Equal(interfaces.ProcessingStatusCompleted, sendModerationProcessing.GetStatus())
		suite.Nil(sendModerationProcessing.GetError())
	}

	for i := 0; i < 4; i++ {
		fetchModerationProcessing := <-notificationChannel
		suite.Equal("fetch_moderation_tg", fetchModerationProcessing.GetBlock().GetId())
		suite.Equal(processingResponse.ProcessingID, fetchModerationProcessing.GetId())

		suite.Equal(interfaces.ProcessingStatusCompleted, fetchModerationProcessing.GetStatus(), i)
		suite.Nil(fetchModerationProcessing.GetError())
	}

	select {
	case <-time.After(500 * time.Millisecond):
	case sideProcessing := <-notificationChannel:
		suite.Fail(
			fmt.Sprintf(
				"Expected notification channel to be empty, but got a value: %s",
				sideProcessing.GetOutput().GetValue().String(),
			),
		)
	}
}
