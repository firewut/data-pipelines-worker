package blocks

import (
	"bytes"
	"context"
	"errors"
	"io"

	openai "github.com/sashabaranov/go-openai"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type ProcessorOpenAIRequestTTS struct {
}

func NewProcessorOpenAIRequestTTS() *ProcessorOpenAIRequestTTS {
	return &ProcessorOpenAIRequestTTS{}
}

func (p *ProcessorOpenAIRequestTTS) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockOpenAIRequestTTSConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockOpenAIRequestTTS).GetBlockConfig(_config)
	userBlockConfig := &BlockOpenAIRequestTTSConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	client := _config.OpenAI.GetClient()
	if client == nil {
		return output, false, errors.New("openAI client is not configured")
	}
	resp, err := client.CreateSpeech(
		ctx,
		openai.CreateSpeechRequest{
			Model:          blockConfig.Model,
			Input:          blockConfig.Text,
			Voice:          blockConfig.Voice,
			ResponseFormat: blockConfig.ResponseFormat,
			Speed:          blockConfig.Speed,
		},
	)
	if err != nil {
		return output, false, err
	}
	defer resp.Close()

	buf, err := io.ReadAll(resp)
	if err != nil {
		return output, false, err
	}

	output = bytes.NewBuffer(buf)

	return output, false, err
}

type BlockOpenAIRequestTTSConfig struct {
	Model          openai.SpeechModel          `yaml:"model" json:"model"`
	Text           string                      `yaml:"text" json:"text"`
	Voice          openai.SpeechVoice          `yaml:"voice" json:"voice"`
	ResponseFormat openai.SpeechResponseFormat `yaml:"response_format" json:"response_format"`
	Speed          float64                     `yaml:"speed" json:"speed"`
}

type BlockOpenAIRequestTTS struct {
	generics.ConfigurableBlock[BlockOpenAIRequestTTSConfig]
	BlockParent
}

var _ interfaces.Block = (*BlockOpenAIRequestTTS)(nil)

func (b *BlockOpenAIRequestTTS) GetBlockConfig(_config config.Config) *BlockOpenAIRequestTTSConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockOpenAIRequestTTSConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockOpenAIRequestTTS() *BlockOpenAIRequestTTS {
	block := &BlockOpenAIRequestTTS{
		BlockParent: BlockParent{
			Id:          "openai_tts_request",
			Name:        "OpenAI Text to Speech",
			Description: "Block to generate audio from text using OpenAI",
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
								"default": "tts-1",
								"enum": ["tts-1"]
							},
							"text": {
								"description": "Text to convert to audio",
								"type": "string",
								"minLength": 10
							},
							"voice": {
								"description": "Voice to use",
								"type": "string",
								"default": "alloy",
								"enum": ["alloy", "echo", "fable", "onyx", "nova", "shimmer"]
							},
							"response_format": {
								"description": "Response format",
								"type": "string",
								"default": "mp3",
								"enum": ["mp3"]
							},
							"speed": {
								"description": "Speed of the audio",
								"type": "number",
								"default": 1.0
							}
						},
						"required": ["text"]
					},
					"output": {
						"description": "OpenAI TTS output",
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

	block.SetProcessor(NewProcessorOpenAIRequestTTS())

	return block
}
