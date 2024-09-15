package unit_test

import (
	"bytes"

	"data-pipelines-worker/types/dataclasses"
)

func (suite *UnitTestSuite) TestGetInputDataByPriority() {
	// Given
	requestPassedInput := map[string]interface{}{
		"url": "https://test-blocks.com/critical",
	}
	inputConfigFromResultInput := map[string]interface{}{
		"url": "https://test-blocks.com/normal",
	}
	inputFromYaml := map[string]interface{}{
		"url": "https://test-blocks.com/low",
	}

	cases := []struct {
		conditions []bool
		expected   map[string]interface{}
	}{
		{
			conditions: []bool{true, false, false},
			expected:   requestPassedInput,
		},
		{
			conditions: []bool{false, true, false},
			expected:   inputConfigFromResultInput,
		},
		{
			conditions: []bool{false, false, true},
			expected:   inputFromYaml,
		},
		{
			conditions: []bool{false, true, true},
			expected:   inputConfigFromResultInput,
		},
		{
			conditions: []bool{true, false, true},
			expected:   requestPassedInput,
		},
		{
			conditions: []bool{true, true, false},
			expected:   requestPassedInput,
		},
		{
			conditions: []bool{true, true, true},
			expected:   requestPassedInput,
		},
	}

	for _, c := range cases {
		// When
		blockInputData := (&dataclasses.BlockData{}).GetInputDataByPriority(
			[]interface{}{
				dataclasses.BlockInputData{
					Condition: c.conditions[0],
					Value: []map[string]interface{}{
						requestPassedInput,
					},
				},
				dataclasses.BlockInputData{
					Condition: c.conditions[1],
					Value: []map[string]interface{}{
						inputConfigFromResultInput,
					},
				},
				dataclasses.BlockInputData{
					Condition: c.conditions[2],
					Value: []map[string]interface{}{
						inputFromYaml,
					},
				},
			},
		)

		// Then
		suite.Equal(1, len(blockInputData))
		suite.Equal(c.expected, blockInputData[0])
	}
}

func (suite *UnitTestSuite) TestGetInputConfigDataNoInputConfig() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "test-block-first-slug",
				"description": "Request Local Resourse",
				"input": {
					"url": "https://test-blocks.com/first"
				}
			}
		]
	}`

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	// When
	firstInputData, err := blocks[0].GetInputConfigData(make(map[string][]*bytes.Buffer, 0))
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
}

func (suite *UnitTestSuite) TestGetInputConfigDataOneDependencyMissing() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "test-block-first-slug",
				"description": "Request Local Resourse",
				"input": {
					"url": "https://test-blocks.com/first"
				}
			},
			{
				"id": "http_request",
				"slug": "test-block-second-slug",
				"description": "Request Result from First Block",
				"input_config": {
					"property": {
						"url": {
							"origin": "test-block-missing-slug"
						}
					}
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"test-block-first-slug": {
			bytes.NewBufferString("https://test-blocks.com/first/response"),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.NotNil(err)
	suite.Contains(err.Error(), "origin test-block-missing-slug not found in pipelineResults")

	// Then
	suite.Empty(firstInputData)
	suite.Empty(secondInputData)
}

func (suite *UnitTestSuite) TestGetInputConfigDataOneDependency() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "test-block-first-slug",
				"description": "Request Local Resourse",
				"input": {
					"url": "https://test-blocks.com/first"
				}
			},
			{
				"id": "http_request",
				"slug": "test-block-second-slug",
				"description": "Request Result from First Block",
				"input_config": {
					"property": {
						"url": {
							"origin": "test-block-first-slug"
						}
					}
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"test-block-first-slug": {
			bytes.NewBufferString("https://test-blocks.com/first/response"),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(
		"https://test-blocks.com/first/response",
		secondInputData[0]["url"],
	)
}

func (suite *UnitTestSuite) TestGetInputConfigDataTwoDependencies() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "test-block-first-slug",
				"description": "Request Local Resourse",
				"input": {
					"url": "https://test-blocks.com/first"
				}
			},
			{
				"id": "http_request",
				"slug": "test-block-second-slug",
				"description": "Request Result from First Block",
				"input_config": {
					"property": {
						"url": {
							"origin": "test-block-first-slug"
						}
					}
				}
			},
			{
				"id": "http_request",
				"slug": "test-block-third-slug",
				"description": "Request Result from First Block",
				"input_config": {
					"property": {
						"url": {
							"origin": "test-block-first-slug"
						},
						"query": {
							"origin": "test-block-second-slug"
						}
					}
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"test-block-first-slug": {
			bytes.NewBufferString("https://test-blocks.com/first/response"),
		},
		"test-block-second-slug": {
			bytes.NewBufferString("Content from second block as description to third block"),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	thirdBlock := blocks[2]
	suite.NotNil(thirdBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	thirdInputData, err := thirdBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)
	suite.NotEmpty(thirdInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(
		"https://test-blocks.com/first/response",
		secondInputData[0]["url"],
	)

	suite.Len(thirdInputData, 1)
	suite.Equal(
		thirdInputData[0],
		map[string]interface{}{
			"url":   "https://test-blocks.com/first/response",
			"query": "Content from second block as description to third block",
		},
	)
}

func (suite *UnitTestSuite) TestGetInputConfigDataJSONPathPlainProperty() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "request-tts-transcription",
				"description": "Request TTS Transcription",
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-image-for-tts-segment",
				"description": "Request an Image for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.language"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"request-tts-transcription": {
			bytes.NewBufferString(suite.GetTestTranscriptionResult()),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 1)
	suite.Equal("english", secondInputData[0]["body"])
}

func (suite *UnitTestSuite) TestGetInputConfigDataJSONPathArrayNthObjectProperty() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "request-tts-transcription",
				"description": "Request TTS Transcription",
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-image-for-tts-segment",
				"description": "Request an Image for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.segments[1].text"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"request-tts-transcription": {
			bytes.NewBufferString(suite.GetTestTranscriptionResult()),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(" Segment two Content", secondInputData[0]["body"])
}

func (suite *UnitTestSuite) TestGetInputConfigDataJSONPathValueAsArray() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "request-tts-transcription",
				"description": "Request TTS Transcription",
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-image-for-tts-segment",
				"description": "Request an Image for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.segments[*].text"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"request-tts-transcription": {
			bytes.NewBufferString(suite.GetTestTranscriptionResult()),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(
		[]interface{}{
			" Segment one Content.",
			" Segment two Content",
		},
		secondInputData[0]["body"],
	)
}

func (suite *UnitTestSuite) TestGetInputConfigDataJSONPathValueAsArrayMultipleDependencies() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "request-tts-transcription",
				"description": "Request TTS Transcription",
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-image-for-tts-segment",
				"description": "Request an Image for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.segments[*].text"
						},
						"query": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.segments[*].end"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"request-tts-transcription": {
			bytes.NewBufferString(suite.GetTestTranscriptionResult()),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(
		[]interface{}{
			" Segment one Content.",
			" Segment two Content",
		},
		secondInputData[0]["body"],
	)
	suite.Equal(
		[]interface{}{1.0, 3.0},
		secondInputData[0]["query"],
	)
}

func (suite *UnitTestSuite) TestGetInputConfigDataTypeNotArrayInputArray() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "request-tts-transcription",
				"description": "Request TTS Transcription",
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-image-for-tts-segment",
				"description": "Request an Image for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.segments[*].text"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "grayscale-image",
				"description": "Grayscale an Image via local service",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-image-for-tts-segment"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"request-tts-transcription": {
			bytes.NewBufferString(suite.GetTestTranscriptionResult()),
		},
		// Warning! The result format is hardcoded for test purposes
		"request-image-for-tts-segment": {
			bytes.NewBufferString("image1"),
			bytes.NewBufferString("image2"),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	thirdBlock := blocks[2]
	suite.NotNil(thirdBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	thirdInputData, err := thirdBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)
	suite.NotEmpty(thirdInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(
		[]interface{}{
			" Segment one Content.",
			" Segment two Content",
		},
		secondInputData[0]["body"],
	)

	suite.Len(thirdInputData, 1)
	suite.Equal(
		"image1",
		thirdInputData[0]["body"],
	)
}

func (suite *UnitTestSuite) TestGetInputConfigDataTypeArrayInputArray() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "request-tts-transcription",
				"description": "Request TTS Transcription",
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-image-for-tts-segment",
				"description": "Request an Image for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.segments[*].text"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-sentiment-for-tts-segment",
				"description": "Request a Duration for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.text"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "grayscale-image",
				"description": "Grayscale an Image via local service",
				"input_config": {
					"type": "array",
					"property": {
						"body": {
							"origin": "request-image-for-tts-segment"
						},
						"query": {
							"origin": "request-sentiment-for-tts-segment"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"request-tts-transcription": {
			bytes.NewBufferString(suite.GetTestTranscriptionResult()),
		},
		// Warning! The result format is hardcoded for test purposes
		"request-image-for-tts-segment": {
			bytes.NewBufferString("image1"),
			bytes.NewBufferString("image2"),
		},
		"request-sentiment-for-tts-segment": {
			bytes.NewBufferString("positive"),
			bytes.NewBufferString("negative"),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	thirdBlock := blocks[2]
	suite.NotNil(thirdBlock)

	fourthBlock := blocks[3]
	suite.NotNil(fourthBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	thirdInputData, err := thirdBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	fourthInputData, err := fourthBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(
		[]interface{}{
			" Segment one Content.",
			" Segment two Content",
		},
		secondInputData[0]["body"],
	)

	suite.NotEmpty(thirdInputData)
	suite.Len(thirdInputData, 1)
	suite.Equal("Segment one Content. Segment two Content", thirdInputData[0]["body"])

	suite.NotEmpty(fourthInputData)
	suite.Len(fourthInputData, 2)
	suite.Equal(
		map[string]interface{}{
			"body":  "image1",
			"query": "positive",
		},
		fourthInputData[0],
	)
	suite.Equal(
		map[string]interface{}{
			"body":  "image2",
			"query": "negative",
		},
		fourthInputData[1],
	)
}

func (suite *UnitTestSuite) TestGetInputConfigDataTypeDefaultInputArray() {
	// Given
	pipelineString := `{
		"slug": "test-pipeline-slug-two-blocks",
		"title": "Test Pipeline",
		"description": "Test Pipeline Description",
		"blocks": [
			{
				"id": "http_request",
				"slug": "request-tts-transcription",
				"description": "Request TTS Transcription",
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-image-for-tts-segment",
				"description": "Request an Image for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.segments[*].text"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "request-sentiment-for-tts-segment",
				"description": "Request a Duration for Text in Transcription Segment",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"jsonPath": "$.text"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			},
			{
				"id": "http_request",
				"slug": "grayscale-image",
				"description": "Grayscale an Image via local service",
				"input_config": {
					"property": {
						"body": {
							"origin": "request-image-for-tts-segment"
						},
						"query": {
							"origin": "request-sentiment-for-tts-segment"
						}
					}
				},
				"input": {
					"url": "https://localhost:8080",
					"method": "POST"
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"request-tts-transcription": {
			bytes.NewBufferString(suite.GetTestTranscriptionResult()),
		},
		// Warning! The result format is hardcoded for test purposes
		"request-image-for-tts-segment": {
			bytes.NewBufferString("image1"),
			bytes.NewBufferString("image2"),
		},
		"request-sentiment-for-tts-segment": {
			bytes.NewBufferString("positive"),
			bytes.NewBufferString("negative"),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	firstBlock := blocks[0]
	suite.NotNil(firstBlock)

	secondBlock := blocks[1]
	suite.NotNil(secondBlock)

	thirdBlock := blocks[2]
	suite.NotNil(thirdBlock)

	fourthBlock := blocks[3]
	suite.NotNil(fourthBlock)

	// When
	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	thirdInputData, err := thirdBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	fourthInputData, err := fourthBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(
		[]interface{}{
			" Segment one Content.",
			" Segment two Content",
		},
		secondInputData[0]["body"],
	)

	suite.NotEmpty(thirdInputData)
	suite.Len(thirdInputData, 1)
	suite.Equal("Segment one Content. Segment two Content", thirdInputData[0]["body"])

	suite.NotEmpty(fourthInputData)
	suite.Len(fourthInputData, 1)
	suite.Equal(
		map[string]interface{}{
			"body":  "image1",
			"query": "positive",
		},
		fourthInputData[0],
	)
}

// func (suite *UnitTestSuite) TestGetInputConfigDataJSONPathOutputConfig() {
// 	// Given
// 	pipelineString := `{
// 		"slug": "test-pipeline-slug-two-blocks",
// 		"title": "Test Pipeline",
// 		"description": "Test Pipeline Description",
// 		"blocks": [
// 			{
// 				"id": "http_request",
// 				"slug": "request-tts-transcription",
// 				"description": "Request TTS Transcription",
// 				"input": {
// 					"url": "https://localhost:8080",
// 					"method": "POST"
// 				}
// 			},
// 			{
// 				"id": "http_request",
// 				"slug": "request-image-for-tts-segment",
// 				"description": "Request an Image for Text in Transcription Segment",
// 				"input_config": {
// 					"property": {
// 						"body": {
// 							"origin": "request-tts-transcription",
// 							"jsonPath": "$.segments[*].text"
// 						}
// 					}
// 				},
// 				"input": {
// 					"url": "https://localhost:8080",
// 					"method": "POST"
// 				},
// 				"output_config": {
// 					"type": "array"
// 				}
// 			}
// 		]
// 	}`

// 	pipelineResults := map[string][]*bytes.Buffer{
// 		"request-tts-transcription": {
// 			bytes.NewBufferString(suite.GetTestTranscriptionResult()),
// 		},
// 	}

// 	pipeline := suite.GetTestPipeline(pipelineString)
// 	suite.NotNil(pipeline)

// 	blocks := pipeline.GetBlocks()
// 	suite.NotEmpty(blocks)

// 	firstBlock := blocks[0]
// 	suite.NotNil(firstBlock)

// 	secondBlock := blocks[1]
// 	suite.NotNil(secondBlock)

// 	// When
// 	firstInputData, err := firstBlock.GetInputConfigData(pipelineResults)
// 	suite.Nil(err)
// 	secondInputData, err := secondBlock.GetInputConfigData(pipelineResults)
// 	suite.Nil(err)

// 	// Then
// 	suite.Empty(firstInputData)
// 	suite.NotEmpty(secondInputData)

// 	suite.Len(secondInputData, 2)
// 	suite.Equal(" Segment one Content", secondInputData[0]["body"])
// 	suite.Equal(" Segment two Content", secondInputData[1]["body"])
// }
