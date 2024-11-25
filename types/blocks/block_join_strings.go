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

type DetectorJoinStrings struct {
	BlockDetectorParent
}

func NewDetectorJoinStrings(
	detectorConfig config.BlockConfigDetector,
) *DetectorJoinStrings {
	return &DetectorJoinStrings{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorJoinStrings) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorJoinStrings struct {
}

func NewProcessorJoinStrings() *ProcessorJoinStrings {
	return &ProcessorJoinStrings{}
}

func (p *ProcessorJoinStrings) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorJoinStrings) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}
func (p *ProcessorJoinStrings) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockJoinStringsConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockJoinStrings).GetBlockConfig(_config)
	userBlockConfig := &BlockJoinStringsConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	stringsList := blockConfig.Strings

	// No need to join if there is only one video
	if len(stringsList) == 1 {
		return bytes.NewBufferString(stringsList[0]), false, false, "", -1, nil
	}

	output = bytes.NewBufferString(
		strings.Join(stringsList, blockConfig.Separator),
	)

	return output, false, false, "", -1, nil
}

type BlockJoinStringsConfig struct {
	Strings   []string `json:"strings" yaml:"-"`
	Separator string   `json:"separator" yaml:"separator"`
}

type BlockJoinStrings struct {
	generics.ConfigurableBlock[BlockJoinStringsConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockJoinStrings)(nil)

func (b *BlockJoinStrings) GetBlockConfig(_config config.Config) *BlockJoinStringsConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockJoinStringsConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockJoinStrings() *BlockJoinStrings {
	block := &BlockJoinStrings{
		BlockParent: BlockParent{
			Id:          "join_strings",
			Name:        "Join Strings",
			Description: "Join multiple strings into one",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"strings": {
								"description": "List of strings to join",
								"type": "array",
								"items": {
									"type": "string"
								},
								"minItems": 1
							},
							"separator": {
								"description": "Separator to use between strings",
								"type": "string",
								"default": ""
							}
						},
						"required": ["strings"]
					},
					"output": {
						"description": "String generated from list of strings",
						"type": ["string", "null"]
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

	block.SetProcessor(NewProcessorJoinStrings())

	return block
}
