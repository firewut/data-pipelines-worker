package blocks

import (
	"bytes"
	"context"

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

func (p *ProcessorStopPipeline) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, error) {
	var (
		output      *bytes.Buffer            = &bytes.Buffer{}
		blockConfig *BlockStopPipelineConfig = &BlockStopPipelineConfig{}
	)
	_data := data.GetInputData().(map[string]interface{})

	// logger := config.GetLogger()

	// Default value from YAML config
	defaultBlockConfig := &BlockStopPipelineConfig{}
	helpers.MapToYAMLStruct(block.GetConfigSection(), defaultBlockConfig)

	// User defined values from data
	userBlockConfig := &BlockStopPipelineConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)

	// Merge the default and user defined maps to BlockConfig
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	stop, err := helpers.EvaluateCondition(blockConfig.Data, blockConfig.Value, blockConfig.Condition)

	return output, stop, err
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

func NewBlockStopPipeline() *BlockStopPipeline {
	_config := config.GetConfig()

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

	block.SetConfigSection(_config.Blocks[block.GetId()].Config)
	block.SetProcessor(NewProcessorStopPipeline())

	return block
}
