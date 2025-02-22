package blocks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorOpenAI struct {
	BlockDetectorParent

	Client *openai.Client
}

func NewDetectorOpenAI(client *openai.Client, detectorConfig config.BlockConfigDetector) *DetectorOpenAI {
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

func (p *ProcessorOpenAIRequestCompletion) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorOpenAIRequestCompletion) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorOpenAIRequestCompletion) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) ([]*bytes.Buffer, bool, bool, string, int, error) {
	output := make([]*bytes.Buffer, 0)
	blockConfig := &BlockOpenAIRequestCompletionConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockOpenAIRequestCompletion).GetBlockConfig(_config)
	userBlockConfig := &BlockOpenAIRequestCompletionConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	client := _config.OpenAI.GetClient()
	if client == nil {
		return output, false, false, "", -1, errors.New("openAI client is not configured")
	}

	messages := make([]openai.ChatCompletionMessage, 0)
	if blockConfig.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: blockConfig.SystemPrompt,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: blockConfig.UserPrompt,
	})

	responseFormat := &openai.ChatCompletionResponseFormat{
		Type: openai.ChatCompletionResponseFormatTypeText,
	}
	if blockConfig.ResponseFormat == "json" {
		responseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:          blockConfig.Model,
			Messages:       messages,
			ResponseFormat: responseFormat,
		},
	)
	if err != nil {
		return output, false, false, "", -1, err
	}

	output = append(
		output,
		bytes.NewBufferString(resp.Choices[0].Message.Content),
	)
	return output, false, false, "", -1, err
}

type BlockOpenAIRequestCompletionConfig struct {
	Model          string `yaml:"model" json:"model"`
	SystemPrompt   string `yaml:"system_prompt" json:"system_prompt"`
	UserPrompt     string `yaml:"user_prompt" json:"user_prompt"`
	ResponseFormat string `yaml:"response_format" json:"response_format"`
}

type BlockOpenAIRequestCompletion struct {
	generics.ConfigurableBlock[BlockOpenAIRequestCompletionConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockOpenAIRequestCompletion)(nil)

func (b *BlockOpenAIRequestCompletion) GetBlockConfig(_config config.Config) *BlockOpenAIRequestCompletionConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockOpenAIRequestCompletionConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockOpenAIRequestCompletion() *BlockOpenAIRequestCompletion {
	block := &BlockOpenAIRequestCompletion{
		BlockParent: BlockParent{
			Id:           "openai_chat_completion",
			Name:         "OpenAI Chat Completion",
			Description:  "Block to perform request to OpenAI's Chat Completion",
			Version:      "1",
			SchemaString: "",
			SchemaPtr:    nil,
			Schema:       nil,
		},
	}

	_config := config.GetConfig()
	defaultBlockConfig := block.GetBlockConfig(_config)

	block.SetSchemaString(
		fmt.Sprintf(`{
				"type": "object",
				"properties": {
					"input": {
						"type": "object",
						"description": "Input parameters",
						"properties": {
							"model": {
								"description": "Model to use",
								"type": "string",
								"default": "%s",
								"enum": ["%s"]
							},
							"system_prompt": {
								"description": "Prompt for OpenAI agent",
								"type": "string",
								"default": "%s"
							},
							"user_prompt": {
								"description": "Prompt for user request",
								"type": "string",
								"default": "%s"
							},
							"response_format": {
								"description": "Response format. Check that model supports it at https://platform.openai.com/docs/guides/structured-outputs/introduction",
								"type": "string",
								"default": "text",
								"enum": ["text", "json"]
							}
						},
						"required": ["user_prompt"]
					},
					"output": {
						"description": "OpenAI Completion output",
						"type": "string"
					}
				},
				"required": ["input"]
			}`,
			defaultBlockConfig.Model,
			defaultBlockConfig.Model,
			defaultBlockConfig.SystemPrompt,
			defaultBlockConfig.UserPrompt,
		),
	)

	if err := block.ApplySchema(block.GetSchemaString()); err != nil {
		panic(err)
	}

	block.SetProcessor(NewProcessorOpenAIRequestCompletion())

	return block
}
