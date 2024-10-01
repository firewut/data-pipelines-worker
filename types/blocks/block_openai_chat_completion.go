package blocks

import (
	"bytes"
	"context"
	"fmt"

	gjm "github.com/firewut/go-json-map"
	openai "github.com/sashabaranov/go-openai"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

// TODO: Add Detector for OpenAI API Access respecting the Authorization Token

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
	var output *bytes.Buffer

	config := config.GetConfig()

	_data := data.GetData().(map[string]interface{})
	model_value, err := gjm.GetProperty(_data, "input.model")
	if err != nil {
		return nil, false, err
	}
	prompt_value, err := gjm.GetProperty(_data, "input.prompt")
	if err != nil {
		return nil, false, err
	}

	client := openai.NewClient(config.OpenAI.Token)
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model_value.(string),
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt_value.(string),
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return output, false, err
	}

	fmt.Println(resp.Choices[0].Message.Content)

	return output, false, nil
}

type BlockOpenAIRequestCompletion struct {
	BlockParent
}

func NewBlockOpenAIRequestCompletion() *BlockOpenAIRequestCompletion {
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

	block.SetProcessor(NewProcessorOpenAIRequestCompletion())

	return block
}
