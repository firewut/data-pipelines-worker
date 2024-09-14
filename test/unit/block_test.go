package unit_test

import (
	"bytes"
	"data-pipelines-worker/types/dataclasses"
)

func (suite *UnitTestSuite) TestGetInputDataFromConfigNoInputConfig() {
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
	firstInputData, err := blocks[0].GetInputDataFromConfig(make(map[string][]*bytes.Buffer, 0))
	suite.Nil(err)

	// Then
	suite.Empty(firstInputData)
}

func (suite *UnitTestSuite) TestGetInputDataFromConfigOneDependencyMissing() {
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
	firstInputData, err := firstBlock.GetInputDataFromConfig(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputDataFromConfig(pipelineResults)
	suite.NotNil(err)
	suite.Contains(err.Error(), "origin test-block-missing-slug not found in pipelineResults")

	// Then
	suite.Empty(firstInputData)
	suite.Empty(secondInputData)
}

func (suite *UnitTestSuite) TestGetInputDataFromConfigOneDependency() {
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
	firstInputData, err := firstBlock.GetInputDataFromConfig(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputDataFromConfig(pipelineResults)
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

func (suite *UnitTestSuite) TestGetInputDataFromConfigTwoDependencies() {
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
	firstInputData, err := firstBlock.GetInputDataFromConfig(pipelineResults)
	suite.Nil(err)
	secondInputData, err := secondBlock.GetInputDataFromConfig(pipelineResults)
	suite.Nil(err)
	thirdInputData, err := thirdBlock.GetInputDataFromConfig(pipelineResults)
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

	suite.Len(thirdInputData, 2)
	suite.Equal(
		"https://test-blocks.com/first/response",
		thirdInputData[0]["url"],
	)
	suite.Equal(
		"Content from second block as description to third block",
		thirdInputData[1]["query"],
	)
}

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

	blockInputData := (&dataclasses.BlockData{}).GetInputDataByPriority(
		[]interface{}{
			dataclasses.BlockInputData{
				Condition: true,
				Value: []map[string]interface{}{
					requestPassedInput,
				},
			},
			dataclasses.BlockInputData{
				Condition: true,
				Value: []map[string]interface{}{
					inputConfigFromResultInput,
				},
			},
			dataclasses.BlockInputData{
				Condition: true,
				Value: []map[string]interface{}{
					inputFromYaml,
				},
			},
		},
	)
	suite.Equal(1, len(blockInputData))
	suite.Equal(requestPassedInput, blockInputData[0])
}
