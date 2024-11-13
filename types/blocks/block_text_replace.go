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

type DetectorTextReplace struct {
	BlockDetectorParent
}

func NewDetectorTextReplace(
	detectorConfig config.BlockConfigDetector,
) *DetectorTextReplace {
	return &DetectorTextReplace{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorTextReplace) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorTextReplace struct {
}

func NewProcessorTextReplace() *ProcessorTextReplace {
	return &ProcessorTextReplace{}
}

func (p *ProcessorTextReplace) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorTextReplace) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorTextReplace) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockTextReplaceConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockTextReplace).GetBlockConfig(_config)
	userBlockConfig := &BlockTextReplaceConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	// Trim spaces from the beginning of the strings
	//    spaces added automatically by Transcription service
	text := strings.TrimLeft(blockConfig.Text, " ")
	_old := strings.TrimLeft(blockConfig.Old, " ")
	_new := strings.TrimLeft(blockConfig.New, " ")
	_prefix := strings.TrimLeft(blockConfig.Prefix, " ")
	_suffix := strings.TrimLeft(blockConfig.Suffix, " ")

	wrapped := _prefix + _new + _suffix
	output = bytes.NewBufferString(
		strings.ReplaceAll(text, _old, wrapped),
	)

	return output, false, false, "", -1, nil
}

type BlockTextReplaceConfig struct {
	Text   string `yaml:"-" json:"text"`
	Old    string `yaml:"-" json:"old"`
	New    string `yaml:"-" json:"new"`
	Prefix string `yaml:"-" json:"prefix"`
	Suffix string `yaml:"-" json:"suffix"`
}

type BlockTextReplace struct {
	generics.ConfigurableBlock[BlockTextReplaceConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockTextReplace)(nil)

func (b *BlockTextReplace) GetBlockConfig(_config config.Config) *BlockTextReplaceConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockTextReplaceConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockTextReplace() *BlockTextReplace {
	block := &BlockTextReplace{
		BlockParent: BlockParent{
			Id:          "text_replace",
			Name:        "Text Replace",
			Description: "Replace text in a string",
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
							"old": {
								"description": "Text to replace",
								"type": "string",
								"minLength": 1
							},	
							"new": {
								"description": "Text to replace with",
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
						"required": ["text", "old", "new"]
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

	block.SetProcessor(NewProcessorTextReplace())

	return block
}
