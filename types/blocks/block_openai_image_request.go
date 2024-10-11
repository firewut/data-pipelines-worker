package blocks

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log"

	openai "github.com/sashabaranov/go-openai"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type ProcessorOpenAIRequestImage struct {
}

func NewProcessorOpenAIRequestImage() *ProcessorOpenAIRequestImage {
	return &ProcessorOpenAIRequestImage{}
}

func (p *ProcessorOpenAIRequestImage) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockOpenAIRequestImageConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockOpenAIRequestImage).GetBlockConfig(_config)
	userBlockConfig := &BlockOpenAIRequestImageConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	client := _config.OpenAI.GetClient()
	if client == nil {
		return output, false, false, errors.New("openAI client is not configured")
	}
	resp, err := client.CreateImage(
		ctx,
		openai.ImageRequest{
			Prompt:         blockConfig.Prompt,
			Model:          openai.CreateImageModelDallE3,
			N:              1,
			Quality:        blockConfig.Quality,
			Size:           blockConfig.Size,
			Style:          blockConfig.Style,
			ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		},
	)
	if err != nil {
		return output, false, false, err
	}

	b64Image := resp.Data[0].B64JSON

	imageData, err := base64.StdEncoding.DecodeString(b64Image)
	if err != nil {
		log.Fatalf("Failed to decode base64 image: %v", err)
	}
	output.Write(imageData)

	return output, false, false, err
}

type BlockOpenAIRequestImageConfig struct {
	Prompt  string `yaml:"prompt" json:"prompt"`
	Quality string `yaml:"quality" json:"quality"`
	Size    string `yaml:"size" json:"size"`
	Style   string `yaml:"style" json:"style"`
}

type BlockOpenAIRequestImage struct {
	generics.ConfigurableBlock[BlockOpenAIRequestImageConfig]
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
							},
							"style": {
								"description": "Style of the image",
								"type": "string"
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
