package blocks

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"

	openai "github.com/sashabaranov/go-openai"
)

type ProcessorOpenAIRequestImage struct {
}

func NewProcessorOpenAIRequestImage() *ProcessorOpenAIRequestImage {
	return &ProcessorOpenAIRequestImage{}
}

func (p *ProcessorOpenAIRequestImage) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorOpenAIRequestImage) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorOpenAIRequestImage) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) ([]*bytes.Buffer, bool, bool, string, int, error) {
	output := make([]*bytes.Buffer, 0)
	blockConfig := &BlockOpenAIRequestImageConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockOpenAIRequestImage).GetBlockConfig(_config)
	userBlockConfig := &BlockOpenAIRequestImageConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	client := _config.OpenAI.GetClient()
	if client == nil {
		return output, false, false, "", -1, errors.New("openAI client is not configured")
	}

	resp, err := client.CreateImage(
		ctx,
		openai.ImageRequest{
			Prompt:         blockConfig.Prompt,
			Model:          openai.CreateImageModelDallE3,
			N:              1,
			Quality:        blockConfig.Quality,
			Size:           blockConfig.Size,
			ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		},
	)
	if err != nil {
		return output, false, false, "", -1, err
	}

	b64Image := resp.Data[0].B64JSON
	imageData, err := base64.StdEncoding.DecodeString(b64Image)
	if err != nil {
		log.Fatalf("Failed to decode base64 image: %v", err)
	}
	output = append(output, bytes.NewBuffer(imageData))

	return output, false, false, "", -1, err
}

type BlockOpenAIRequestImageConfig struct {
	Prompt  string `yaml:"prompt" json:"prompt"`
	Quality string `yaml:"quality" json:"quality"`
	Size    string `yaml:"size" json:"size"`
}

type BlockOpenAIRequestImage struct {
	generics.ConfigurableBlock[BlockOpenAIRequestImageConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockOpenAIRequestImage)(nil)

func (b *BlockOpenAIRequestImage) GetBlockConfig(_config config.Config) *BlockOpenAIRequestImageConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockOpenAIRequestImageConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockOpenAIRequestImage() *BlockOpenAIRequestImage {
	block := &BlockOpenAIRequestImage{
		BlockParent: BlockParent{
			Id:          "openai_image_request",
			Name:        "OpenAI Image Request",
			Description: "Block to generate image from text using OpenAI Dall-e v3",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input": {
						"type": "object",
						"description": "Input parameters",
						"properties": {
							"prompt": {
								"description": "Prompt for the image",
								"type": "string",
								"minLength": 10
							},
							"quality": {
								"description": "Quality of the image",
								"type": "string",
								"default": "standard",
								"enum": ["hd", "standard"]
							},
							"size": {
								"description": "Size of the image",
								"type": "string",
								"default": "1024x1024",
								"enum": ["1024x1024", "1024x1792", "1792x1024"]
							}
						},
						"required": ["prompt"]
					},
					"output": {
						"description": "OpenAI Image output",
						"type": "string",
						"format": "file"
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

	block.SetProcessor(NewProcessorOpenAIRequestImage())

	return block
}
