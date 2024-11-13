package blocks

import (
	"bytes"
	"context"
	"strings"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorTextAddPrefixOrSuffix struct {
	BlockDetectorParent
}

func NewDetectorTextAddPrefixOrSuffix(
	detectorConfig config.BlockConfigDetector,
) *DetectorTextAddPrefixOrSuffix {
	return &DetectorTextAddPrefixOrSuffix{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorTextAddPrefixOrSuffix) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorTextAddPrefixOrSuffix struct {
}

func NewProcessorTextAddPrefixOrSuffix() *ProcessorTextAddPrefixOrSuffix {
	return &ProcessorTextAddPrefixOrSuffix{}
}

func (p *ProcessorTextAddPrefixOrSuffix) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorTextAddPrefixOrSuffix) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorTextAddPrefixOrSuffix) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockTextAddPrefixOrSuffixConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockTextAddPrefixOrSuffix).GetBlockConfig(_config)
	userBlockConfig := &BlockTextAddPrefixOrSuffixConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	// Trim spaces from the beginning of the strings
	//    spaces added automatically by Transcription service
	text := strings.TrimLeft(blockConfig.Text, " ")
	_prefix := strings.TrimLeft(blockConfig.Prefix, " ")
	_suffix := strings.TrimLeft(blockConfig.Suffix, " ")

	output = bytes.NewBufferString(_prefix + text + _suffix)

	return output, false, false, "", -1, nil
}

type BlockTextAddPrefixOrSuffixConfig struct {
	Text   string `yaml:"-" json:"text"`
	Prefix string `yaml:"-" json:"prefix"`
	Suffix string `yaml:"-" json:"suffix"`
}

type BlockTextAddPrefixOrSuffix struct {
	generics.ConfigurableBlock[BlockTextAddPrefixOrSuffixConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockTextAddPrefixOrSuffix)(nil)

func (b *BlockTextAddPrefixOrSuffix) GetBlockConfig(_config config.Config) *BlockTextAddPrefixOrSuffixConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockTextAddPrefixOrSuffixConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockTextAddPrefixOrSuffix() *BlockTextAddPrefixOrSuffix {
	block := &BlockTextAddPrefixOrSuffix{
		BlockParent: BlockParent{
			Id:          "wrap_text",
			Name:        "Text Add Prefix or Suffix",
			Description: "Add a prefix and/or suffix to a text",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"text": {
								"description": "Text to make changes in",
								"type": "string",
								"minLength": 1
							},
							"prefix": {
								"description": "Prefix to add to old text",
								"type": "string"
							},
							"suffix": {
								"description": "Suffix to add to new text",
								"type": "string"
							}
						},
						"required": ["text"]
					},
					"output": {
						"description": "Text with changes",
						"type": "string"
					}
				}
			}`,
			SchemaPtr: nil,
			Schema:    nil,
		},
	}

	if err := block.ApplySchema(block.GetSchemaString()); err != nil {
		panic(err)
	}

	block.SetProcessor(NewProcessorTextAddPrefixOrSuffix())

	return block
}
