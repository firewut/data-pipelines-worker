package blocks

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"time"

	"github.com/disintegration/imaging"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorImageResize struct {
	BlockDetectorParent
}

func NewDetectorImageResize(
	detectorConfig config.BlockConfigDetector,
) *DetectorImageResize {
	return &DetectorImageResize{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorImageResize) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorImageResize struct {
}

func NewProcessorImageResize() *ProcessorImageResize {
	return &ProcessorImageResize{}
}

func (p *ProcessorImageResize) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorImageResize) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorImageResize) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockImageResizeConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockImageResize).GetBlockConfig(_config)
	userBlockConfig := &BlockImageResizeConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	imageBytes, err := helpers.GetValue[[]byte](_data, "image")
	if err != nil {
		return nil, false, false, "", -1, err
	}

	imgBuf := bytes.NewBuffer(imageBytes)
	img, format, err := image.Decode(imgBuf)
	if err != nil {
		return nil, false, false, "", -1, err
	}
	config.GetLogger().Debugf("Image format: %s", format)

	resizedImage := imaging.Resize(img, blockConfig.Width, blockConfig.Height, imaging.Lanczos)

	err = png.Encode(output, resizedImage)
	if err != nil {
		config.GetLogger().Fatalf("Failed to encode PNG image: %v", err)
	}

	return output, false, false, "", -1, nil
}

type BlockImageResizeConfig struct {
	Width           int  `yaml:"width" json:"width"`
	Height          int  `yaml:"height" json:"height"`
	KeepAspectRatio bool `yaml:"keep_aspect_ratio" json:"keep_aspect_ratio"`
}

type BlockImageResize struct {
	generics.ConfigurableBlock[BlockImageResizeConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockImageResize)(nil)

func (b *BlockImageResize) GetBlockConfig(_config config.Config) *BlockImageResizeConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockImageResizeConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockImageResize() *BlockImageResize {
	block := &BlockImageResize{
		BlockParent: BlockParent{
			Id:          "image_resize",
			Name:        "Image Resize",
			Description: "Resize Image",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"image": {
								"description": "Image to add text to",
								"type": "string",
								"format": "file"
							},
							"width": {
								"description": "Width of the image",
								"type": "integer",
								"default": 100
							},
							"height": {
								"description": "Height of the image",
								"type": "integer",
								"default": 100
							},
							"keep_aspect_ratio": {
								"description": "Keep aspect ratio",
								"type": "boolean",
								"default": true
							}
						},
						"required": ["image"]
					},
					"output": {
						"description": "Resized image",
						"type": ["string", "null"],
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

	block.SetProcessor(NewProcessorImageResize())

	return block
}
