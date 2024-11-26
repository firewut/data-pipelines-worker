package unit_test

import (
	"bytes"
	"reflect"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
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

func (suite *UnitTestSuite) TestGetInputDataByPriorityMerged() {
	// Given
	requestPassedInput := map[string]interface{}{
		"url": "https://test-blocks.com/critical",
	}
	inputConfigFromResultInput := map[string]interface{}{
		"url": "https://test-blocks.com/normal",
	}
	inputFromYaml := map[string]interface{}{
		"url":    "https://test-blocks.com/low",
		"method": "POST",
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
			expected: map[string]interface{}{
				"url":    "https://test-blocks.com/normal",
				"method": "POST",
			},
		},
		{
			conditions: []bool{true, false, true},
			expected: map[string]interface{}{
				"url":    "https://test-blocks.com/critical",
				"method": "POST",
			},
		},
		{
			conditions: []bool{true, true, false},
			expected:   requestPassedInput,
		},
		{
			conditions: []bool{true, true, true},
			expected: map[string]interface{}{
				"url":    "https://test-blocks.com/critical",
				"method": "POST",
			},
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

func (suite *UnitTestSuite) TestMergeMaps() {
	// Given
	cases := []struct {
		left     []map[string]interface{}
		right    []map[string]interface{}
		expected []map[string]interface{}
	}{
		{
			left: []map[string]interface{}{
				{
					"key1": "value1",
				},
			},
			right: []map[string]interface{}{
				{
					"key2": "value2",
				},
			},
			expected: []map[string]interface{}{
				{
					"key1": "value1",
					"key2": "value2",
				},
			},
		},
		{
			left: []map[string]interface{}{
				{
					"url":    "https://test-blocks.com/first",
					"method": "GET",
				},
			},
			right: []map[string]interface{}{
				{
					"method": "POST",
				},
			},
			expected: []map[string]interface{}{
				{
					"url":    "https://test-blocks.com/first",
					"method": "GET",
				},
				{
					"url":    "https://test-blocks.com/first",
					"method": "POST",
				},
			},
		},
		{
			left: []map[string]interface{}{
				{
					"prompt":  "On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.",
					"quality": "hd",
					"size":    "1024x1792",
				},
				{
					"prompt": "This marked the start of their legendary musical journey, leading to global fame.",
				},
			},
			right: []map[string]interface{}{
				{
					"prompt": "Interestingly, John Lennon's harmonica playing added a distinct touch",
				},
				{
					"prompt": "propelling them toward unprecedented stardom in the music industry.",
				},
			},
			expected: []map[string]interface{}{
				{
					"prompt":  "On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.",
					"quality": "hd",
					"size":    "1024x1792",
				},
				{
					"prompt":  "This marked the start of their legendary musical journey, leading to global fame.",
					"quality": "hd",
					"size":    "1024x1792",
				},
				{
					"prompt":  "Interestingly, John Lennon's harmonica playing added a distinct touch",
					"quality": "hd",
					"size":    "1024x1792",
				},
				{
					"prompt":  "propelling them toward unprecedented stardom in the music industry.",
					"quality": "hd",
					"size":    "1024x1792",
				},
			},
		},
	}

	for _, c := range cases {
		// When
		result := dataclasses.MergeMaps(append(c.left, c.right...))

		// Then
		suite.Equal(c.expected, result)
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
	firstInputData, _, _, err := blocks[0].GetInputConfigData(make(map[string][]*bytes.Buffer, 0))
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.NotNil(err, err)
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
				},
				"input": {
					"method": "POST"
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 1)
	suite.Equal(secondInputData[0],
		map[string]interface{}{
			"url": "https://test-blocks.com/first/response",
		},
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	thirdInputData, _, _, err := thirdBlock.GetInputConfigData(pipelineResults)
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
							"json_path": "$.language"
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
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
							"json_path": "$.segments[1].text"
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
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
							"json_path": "$.segments[*].text"
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
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
							"json_path": "$.segments[*].text"
						},
						"query": {
							"origin": "request-tts-transcription",
							"json_path": "$.segments[*].end"
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
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
							"json_path": "$.segments[*].text"
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	thirdInputData, _, _, err := thirdBlock.GetInputConfigData(pipelineResults)
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
							"json_path": "$.segments[*].text"
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
							"json_path": "$.text"
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	thirdInputData, _, _, err := thirdBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	fourthInputData, _, _, err := fourthBlock.GetInputConfigData(pipelineResults)
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
					"type": "array",
					"property": {
						"body": {
							"origin": "request-tts-transcription",
							"json_path": "$.segments[*].text"
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
							"json_path": "$.text"
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
	firstInputData, _, _, err := firstBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	secondInputData, _, _, err := secondBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	thirdInputData, _, _, err := thirdBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)
	fourthInputData, _, _, err := fourthBlock.GetInputConfigData(pipelineResults)
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
	suite.NotEmpty(secondInputData)

	suite.Len(secondInputData, 2)
	suite.Equal(
		[]map[string]interface{}{
			{"body": " Segment one Content."},
			{"body": " Segment two Content"},
		},
		secondInputData,
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

func (suite *UnitTestSuite) TestOpenAIPipeline() {
	// Given
	pipelineString := `{
		"slug": "openai-test",
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
						"audio": {
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
	}`
	pipelineResults := map[string][]*bytes.Buffer{
		"get-event-text": {
			bytes.NewBufferString(
				`On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry.`,
			),
		},
		"get-event-tts": {
			bytes.NewBufferString("tts-binary-content"),
		},
		"get-event-transcription": {
			bytes.NewBufferString(
				`{"task":"transcribe","language":"english","duration":21.690000534057617,"segments":[{"id":0,"seek":0,"start":0,"end":8.140000343322754,"text":" On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.","tokens":[50364,1282,7617,1025,11,39498,11,264,1002,390,5680,3105,382,264,38376,4736,641,13828,2167,294,264,7051,13,50771],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":1,"seek":0,"start":8.140000343322754,"end":12.899999618530273,"text":" This marked the start of their legendary musical journey, leading to global fame.","tokens":[50771,639,12658,264,722,295,641,16698,9165,4671,11,5775,281,4338,16874,13,51009],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":2,"seek":0,"start":12.899999618530273,"end":16.739999771118164,"text":" Interestingly, John Lennon's harmonica playing added a distinct touch,","tokens":[51009,30564,11,2619,441,1857,266,311,14750,2262,2433,3869,257,10644,2557,11,51201],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false},{"id":3,"seek":0,"start":16.739999771118164,"end":20.540000915527344,"text":" propelling them toward unprecedented stardom in the music industry.","tokens":[51201,25577,2669,552,7361,21555,342,515,298,294,264,1318,3518,13,51391],"temperature":0,"avg_logprob":-0.29121363162994385,"compression_ratio":1.4727272987365723,"no_speech_prob":0.00016069135745055974,"transient":false}],"words":null,"text":"On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry."}`,
			),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	imageRequestblock := blocks[3]
	suite.NotNil(imageRequestblock)

	// When
	transcriptions, _, _, err := imageRequestblock.GetInputConfigData(pipelineResults)

	// Then
	suite.Nil(err)
	suite.NotEmpty(transcriptions)
	suite.Equal(reflect.Slice, reflect.TypeOf(transcriptions).Kind())
	suite.Len(transcriptions, 4)

	suite.Equal([]map[string]interface{}{
		{"prompt": " On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK."},
		{"prompt": " This marked the start of their legendary musical journey, leading to global fame."},
		{"prompt": " Interestingly, John Lennon's harmonica playing added a distinct touch,"},
		{"prompt": " propelling them toward unprecedented stardom in the music industry."},
	}, transcriptions)
}

func (suite *UnitTestSuite) TestPipelineJoinVideos() {
	// Given
	pipelineString := `{
		"slug": "openai-test",
		"title": "Youtube video generation pipeline from prompt",
		"description": "Generates videos for youtube Channel <CHANNEL>. Uses Prompt in the Block.",
		"blocks": [
			{
				"id": "video_from_image",
				"slug": "generate-videos-from-images-and-transcriptions",
				"description": "Generate video from Images and Transcription Segments",
				"input_config": {
					"type": "array", 
					"parallel": true,
					"property": {
						"start": {
							"origin": "get-event-transcription",
							"json_path": "$.segments[*].start"
						},
						"end": {
							"origin": "get-event-transcription",
							"json_path": "$.segments[*].end"
						},
						"image": {
							"origin": "add-text-to-event-images"
						}
					}
				}
			},
			{
				"id": "join_videos",
				"slug": "join-videos-from-array",
				"description": "Join Videos from Array of Videos",
				"input_config": {
					"property": {
						"videos": {
							"origin": "generate-videos-from-images-and-transcriptions",
							"array_input": true
						}
					}
				}
			}
		]
	}`

	imageWidth := 2
	imageHeight := 2

	images := []bytes.Buffer{
		factories.GetPNGImageBuffer(imageWidth, imageHeight),
		factories.GetPNGImageBuffer(imageWidth, imageHeight),
	}
	videos := make([]*bytes.Buffer, len(images))
	for i, image := range images {
		videoBlock := blocks.NewBlockVideoFromImage()
		_data := &dataclasses.BlockData{
			Id:   "video_from_image",
			Slug: "video-from-image",
			Input: map[string]interface{}{
				"image":  image.Bytes(),
				"start":  0.0,
				"end":    0.1,
				"fps":    1,
				"preset": "veryfast",
				"crf":    23,
			},
		}
		_data.SetBlock(videoBlock)
		_result, _stop, _, _, _, err := videoBlock.Process(
			suite.GetContextWithcancel(),
			blocks.NewProcessorVideoFromImage(),
			_data,
		)

		suite.NotNil(_result)
		suite.False(_stop)
		suite.Nil(err)

		videos[i] = _result[0]
	}

	pipelineResults := map[string][]*bytes.Buffer{
		"generate-videos-from-images-and-transcriptions": videos,
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	videosJoinBlock := blocks[1]
	suite.NotNil(videosJoinBlock)
	suite.Equal("join_videos", videosJoinBlock.GetId())

	// When
	inputData, _, _, err := videosJoinBlock.GetInputConfigData(pipelineResults)

	// Then
	suite.Nil(err)
	suite.NotEmpty(inputData)
	suite.Equal(1, len(inputData))
	suite.Equal(2, len(inputData[0]["videos"].([]interface{})))
}

func (suite *UnitTestSuite) TestPipelineJoinStrings() {
	// Given
	pipelineString := `{
		"slug": "openai-test",
		"title": "Youtube video generation pipeline from prompt",
		"description": "Generates videos for youtube Channel <CHANNEL>. Uses Prompt in the Block.",
		"blocks": [
			{
				"id": "openai_chat_completion",
				"slug": "analyze-user-input",
				"description": "Get a text from OpenAI Chat Completion",
				"input": {
					"response_format": "json",
					"model": "gpt-4o-2024-08-06",
					"user_prompt": "Your time is limited, so don't waste it living someone else's life"
				}
			},
			{
				"id": "openai_chat_completion",
				"slug": "get-summary-from-image-prompt",
				"description": "Get summary from prompt used for Image generation using OpenAI Chat Completion",
				"input_config": {
					"type": "array",
					"property": {
						"user_prompt": {
							"origin": "analyze-user-input",
							"json_path": "$.background_suggestions[*]"
						}
					}
				},
				"input": {
					"model": "gpt-4o-2024-08-06",
					"system_prompt": "..."
				}
			},
			{
				"id": "join_strings",
				"slug": "make-text-for-audio-generation",
				"description": "Make a text for audio generation from the Summaries",
				"input_config": {
					"property": {
						"strings": {
							"origin": "get-summary-from-image-prompt",
							"array_input": true
						}
					}
				},
				"input": {
					"separator": ". "
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"analyze-user-input": {
			bytes.NewBufferString(`{ 
				"text": "Your time is limited, so do not waste it living someone else life", 
				"color_palette": ["#FFA726", "#F57C00", "#FFEB3B", "#E64A19"], 
				"typography": "bold", 
				"background_suggestions": [
					"a sunrise over a mountain, symbolizing new beginnings and self-discovery", 
					"an empty road stretching into the horizon, representing a journey towards personal fulfillment", 
					"an open book set on a table, signifying the importance of writing your own story"
				] 
			}`),
		},
		"get-summary-from-image-prompt": {
			bytes.NewBufferString(`a sunrise over a mountain, symbolizing new beginnings and self-discovery`),
			bytes.NewBufferString(`an empty road stretching into the horizon, representing a journey towards personal fulfillment`),
			bytes.NewBufferString(`an open book set on a table, signifying the importance of writing your own story`),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	stringsJoinBlock := blocks[2]
	suite.NotNil(stringsJoinBlock)
	suite.Equal("join_strings", stringsJoinBlock.GetId())

	// When
	inputData, _, _, err := stringsJoinBlock.GetInputConfigData(pipelineResults)

	// Then
	suite.Nil(err)
	suite.NotEmpty(inputData)
	suite.Equal(1, len(inputData))
	suite.Equal(3, len(inputData[0]["strings"].([]interface{})))
}

func (suite *UnitTestSuite) TestPipelineJoinStringsJSONPath() {
	// Given
	pipelineString := `{
		"slug": "openai-test",
		"title": "Youtube video generation pipeline from prompt",
		"description": "Generates videos for youtube Channel <CHANNEL>. Uses Prompt in the Block.",
		"blocks": [
			{
				"id": "audio_chunk",
				"slug": "chunk-podcast",
				"description": "Chunk the podcast into smaller parts for processing",
				"input_config": {
					"property": {
						"audio": {
							"origin": "upload-podcast"
						}
					}
				},
				"input": {
					"duration": "10m"
				}
			},
			{
				"id": "openai_transcription_request",
				"slug": "get-podcast-transcription",
				"description": "Make a request to OpenAI TTS API to convert audio to transcription",
				"input_config": {
					"type": "array",
					"property": {
						"audio": {
							"origin": "chunk-podcast"
						}
					}
				},
				"input": {
					"format": "text"
				}
			},
			{
				"id": "join_strings",
				"slug": "join-transcription-chunks",
				"description": "Join the transcription chunks into a single text",
				"input_config": {
					"property": {
						"strings": {
							"origin": "get-podcast-transcription",
							"json_path": "$[*].text",
							"array_input": true
						}
					}
				},
				"input": {
					"separator": " "
				}
			},
			{
				"id": "openai_chat_completion",
				"slug": "get-summary-of-a-podcast",
				"description": "Gets summary of a podcast from OpenAI Chat Completion",
				"input_config": {
					"property": {
						"user_prompt": {
							"origin": "join-transcription-chunks"
						}
					}
				},
				"input": {
					"model": "gpt-4o-2024-08-06",
					"system_prompt": "Please summarize the following podcast episode transcript in a concise and engaging way, highlighting the key topics, themes, and insights discussed. The summary should be approximately 200 words long and suitable for use as a description on podcast platforms. Avoid technical jargon unless necessary and ensure the tone is professional yet accessible."
				}
			}
		]
	}`

	pipelineResults := map[string][]*bytes.Buffer{
		"get-podcast-transcription": {
			bytes.NewBufferString(`{"task":"","language":"","duration":0,"segments":null,"words":null,"text":"1a"}`),
			bytes.NewBufferString(`{"task":"","language":"","duration":0,"segments":null,"words":null,"text":"2b"}`),
			bytes.NewBufferString(`{"task":"","language":"","duration":0,"segments":null,"words":null,"text":"3c"}`),
		},
	}

	pipeline := suite.GetTestPipeline(pipelineString)
	suite.NotNil(pipeline)

	blocks := pipeline.GetBlocks()
	suite.NotEmpty(blocks)

	stringsJoinBlock := blocks[2]
	suite.NotNil(stringsJoinBlock)
	suite.Equal("join_strings", stringsJoinBlock.GetId())

	// When
	inputData, _, _, err := stringsJoinBlock.GetInputConfigData(pipelineResults)

	// Then
	suite.Nil(err)
	suite.NotEmpty(inputData)
	suite.Equal(1, len(inputData))
	suite.Equal(3, len(inputData[0]["strings"].([]interface{})))
	suite.EqualValues(
		[]interface{}{"1a", "2b", "3c"},
		inputData[0]["strings"],
	)
}
