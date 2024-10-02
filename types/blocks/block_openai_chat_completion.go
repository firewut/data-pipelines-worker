package blocks

import (
	"bytes"
	"context"
	"errors"

	openai "github.com/sashabaranov/go-openai"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorOpenAI struct {
	BlockDetectorParent

	Client *openai.Client
}

func NewDetectorOpenAI(
	client *openai.Client,
	detectorConfig config.BlockConfigDetector,
) *DetectorOpenAI {
	return &DetectorOpenAI{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
		Client:              client,
	}
}

func (d *DetectorOpenAI) Detect() bool {
	d.Lock()
	defer d.Unlock()

	if d.Client == nil {
		return false
	}

	_, err := d.Client.ListModels(context.Background())
	return err == nil
}

type ProcessorOpenAIRequestCompletion struct {
}

func NewProcessorOpenAIRequestCompletion() *ProcessorOpenAIRequestCompletion {
	return &ProcessorOpenAIRequestCompletion{}
}

func (p *ProcessorOpenAIRequestCompletion) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, error) {
	var (
		output      *bytes.Buffer                       = &bytes.Buffer{}
		blockConfig *BlockOpenAIRequestCompletionConfig = &BlockOpenAIRequestCompletionConfig{}
	)
	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := &BlockOpenAIRequestCompletionConfig{}
	helpers.MapToYAMLStruct(block.GetConfigSection(), defaultBlockConfig)

	// User defined values from data
	userBlockConfig := &BlockOpenAIRequestCompletionConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)

	// Merge the default and user defined maps to BlockConfig
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	client := _config.OpenAI.GetClient()
	if client == nil {
		return output, false, errors.New("openAI client is not configured")
	}
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: blockConfig.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: blockConfig.Prompt,
				},
			},
		},
	)

	output = bytes.NewBufferString(resp.Choices[0].Message.Content)

	return output, false, err
}

type BlockOpenAIRequestCompletionConfig struct {
	Model  string `yaml:"model" json:"model"`
	Prompt string `yaml:"prompt" json:"prompt"`
}

type BlockOpenAIRequestCompletion struct {
	BlockParent
}

func NewBlockOpenAIRequestCompletion() *BlockOpenAIRequestCompletion {
	_config := config.GetConfig()

	block := &BlockOpenAIRequestCompletion{
		BlockParent: BlockParent{
			Id:          "openai_chat_completion",
			Name:        "OpenAI Chat Completion",
			Description: "Block to perform request to OpenAI's Chat Completion",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input": {
						"type": "object",
						"description": "Input parameters",
						"properties": {
							"model": {
								"description": "Model to use",
								"type": "string",
								"default": "gpt-4o",
								"enum": ["gpt-4o"]
							},
							"prompt": {
								"description": "Prompt to generate completion from",
								"type": "string"
							}
						},
						"required": ["model", "prompt"]
					},
					"output": {
						"description": "OpenAI Completion output",
						"type": "string"
					}
				},
				"required": ["input"]
			}`,
			SchemaPtr: nil,
			Schema:    nil,
		},
	}

	if err := block.ApplySchema(block.GetSchemaString()); err != nil {
		panic(err)
	}

	block.SetConfigSection(_config.Blocks[block.GetId()].Config)
	block.SetProcessor(NewProcessorOpenAIRequestCompletion())

	return block
}
