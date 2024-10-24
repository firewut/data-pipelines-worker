package blocks

import (
	"bytes"
	"context"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorStopPipeline struct {
	BlockDetectorParent
}

func NewDetectorStopPipeline(
	detectorConfig config.BlockConfigDetector,
) *DetectorStopPipeline {
	return &DetectorStopPipeline{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorStopPipeline) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorStopPipeline struct {
}

func NewProcessorStopPipeline() *ProcessorStopPipeline {
	return &ProcessorStopPipeline{}
}

func (p *ProcessorStopPipeline) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorStopPipeline) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorStopPipeline) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockStopPipelineConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockStopPipeline).GetBlockConfig(_config)
	userBlockConfig := &BlockStopPipelineConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	stop, err := helpers.EvaluateCondition(blockConfig.Data, blockConfig.Value, blockConfig.Condition)

	return output, stop, false, "", -1, err
}

type BlockStopPipelineConfig struct {
	Stop      bool   `yaml:"stop" json:"-"`
	Data      string `yaml:"data" json:"data"`
	Condition string `yaml:"condition" json:"condition"`
	Value     string `yaml:"value" json:"value"`
}

type BlockStopPipeline struct {
	BlockParent
	Config *BlockStopPipelineConfig
}

var _ interfaces.Block = (*BlockStopPipeline)(nil)

func (b *BlockStopPipeline) GetBlockConfig(_config config.Config) *BlockStopPipelineConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockStopPipelineConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockStopPipeline() *BlockStopPipeline {
	block := &BlockStopPipeline{
		BlockParent: BlockParent{
			Id:          "stop_pipeline",
			Name:        "Pipeline Stop",
			Description: "Stop the pipeline if a condition is met",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"data": {
								"description": "Source value to compare",
								"type": "string"
							},
							"condition": {
								"description": "Condition to execute",
								"type": "string",
								"enum": ["==", "!=", ">", "<", ">=", "<="]
							},
							"value": {
								"description": "Property key to check value from",
								"type": "string"
							}
						},
						"required": ["data", "condition", "value"]
					},
					"output": {
						"description": "Condition to stop the pipeline",
						"type": "null"
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

	block.SetProcessor(NewProcessorStopPipeline())

	return block
}
