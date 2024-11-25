package blocks

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorUploadFile struct {
	BlockDetectorParent
}

func NewDetectorUploadFile(
	detectorConfig config.BlockConfigDetector,
) *DetectorUploadFile {
	return &DetectorUploadFile{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorUploadFile) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorUploadFile struct {
}

func NewProcessorUploadFile() *ProcessorUploadFile {
	return &ProcessorUploadFile{}
}

func (p *ProcessorUploadFile) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorUploadFile) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorUploadFile) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockUploadFileConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockUploadFile).GetBlockConfig(_config)
	userBlockConfig := &BlockUploadFileConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	fileBytes, err := helpers.GetValue[[]byte](_data, "file")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	if len(fileBytes) == 0 {
		return nil, false, false, "", -1, fmt.Errorf(
			"file is empty",
		)
	}

	if _, err := output.Write(fileBytes); err != nil {
		return nil, false, false, "", -1, err
	}

	return output, false, false, "", -1, err
}

type BlockUploadFileConfig struct {
}

type BlockUploadFile struct {
	generics.ConfigurableBlock[BlockUploadFileConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockUploadFile)(nil)

func (b *BlockUploadFile) GetBlockConfig(_config config.Config) *BlockUploadFileConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockUploadFileConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockUploadFile() *BlockUploadFile {
	block := &BlockUploadFile{
		BlockParent: BlockParent{
			Id:          "upload_file",
			Name:        "Upload File",
			Description: "Upload a file. This block is used to upload a file to the server and store it.",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"file": {
								"type": "string",
								"description": "File to upload",
								"format": "file",
								"minLength": 1
							}
						},
						"required": ["file"]
					},
					"output": {
						"description": "Uploaded file",
						"type": "string",
						"format": "file"
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

	block.SetProcessor(NewProcessorUploadFile())

	return block
}
