package blocks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type ProcessorOpenAIRequestTranscription struct {
}

func NewProcessorOpenAIRequestTranscription() *ProcessorOpenAIRequestTranscription {
	return &ProcessorOpenAIRequestTranscription{}
}

func (p *ProcessorOpenAIRequestTranscription) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorOpenAIRequestTranscription) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorOpenAIRequestTranscription) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) ([]*bytes.Buffer, bool, bool, string, int, error) {
	output := make([]*bytes.Buffer, 0)
	blockConfig := &BlockOpenAIRequestTranscriptionConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockOpenAIRequestTranscription).GetBlockConfig(_config)
	userBlockConfig := &BlockOpenAIRequestTranscriptionConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	client := _config.OpenAI.GetClient()
	if client == nil {
		return output, false, false, "", -1, errors.New("openAI client is not configured")
	}

	audioBytes, err := helpers.GetValue[[]byte](_data, "audio")
	if err != nil {
		return nil, false, false, "", -1, err
	}

	resp, err := client.CreateTranscription(
		ctx,
		openai.AudioRequest{
			Model:    blockConfig.Model,
			Language: blockConfig.Language,
			Format:   blockConfig.Format,
			FilePath: fmt.Sprintf("%s.mp3", block.GetId()),
			Reader:   bytes.NewReader(audioBytes),
		},
	)
	if err != nil {
		return output, false, false, "", -1, err
	}

	// Encode response to JSON
	jsonData, err := json.Marshal(resp)
	if err != nil {
		return output, false, false, "", -1, err
	}

	output = append(output, bytes.NewBuffer(jsonData))

	return output, false, false, "", -1, err
}

type BlockOpenAIRequestTranscriptionConfig struct {
	Model    string                     `yaml:"model" json:"model"`
	Language string                     `yaml:"language" json:"language"`
	Format   openai.AudioResponseFormat `yaml:"format" json:"format"`
}

type BlockOpenAIRequestTranscription struct {
	generics.ConfigurableBlock[BlockOpenAIRequestTranscriptionConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockOpenAIRequestTranscription)(nil)

func (b *BlockOpenAIRequestTranscription) GetBlockConfig(_config config.Config) *BlockOpenAIRequestTranscriptionConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockOpenAIRequestTranscriptionConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockOpenAIRequestTranscription() *BlockOpenAIRequestTranscription {
	block := &BlockOpenAIRequestTranscription{
		BlockParent: BlockParent{
			Id:          "openai_transcription_request",
			Name:        "OpenAI Audio Transcription",
			Description: "Block to Transcribe audio to JSON using OpenAI. File size limit is 20MB and duration limit is 10 minutes.",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input": {
						"type": "object",
						"description": "Input parameters",
						"properties": {
							"audio": {
								"description": "Audio file to transcribe",
								"type": "string",
								"format": "file"
							},
							"model": {
								"description": "Model to use",
								"type": "string",
								"default": "whisper-1",
								"enum": ["whisper-1"]
							},
							"format": {
								"description": "Response format",
								"type": "string",
								"default": "verbose_json",
								"enum": ["verbose_json", "text"]
							},
							"language": {
								"description": "Language of the audio",
								"type": "string",
								"default": "en",
								"enum": ["en"]
							}
						},
						"required": ["audio"]
					},
					"output": {
						"description": "OpenAI Transcription output",
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

	block.SetProcessor(NewProcessorOpenAIRequestTranscription())

	return block
}
