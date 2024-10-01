package blocks

import (
	"bytes"
	"context"
	"image"
	"image/png"

	"github.com/disintegration/imaging"

	"data-pipelines-worker/types/config"
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

func (p *ProcessorImageResize) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, error) {
	var (
		output      *bytes.Buffer           = &bytes.Buffer{}
		blockConfig *BlockImageResizeConfig = &BlockImageResizeConfig{}
	)
	_data := data.GetInputData().(map[string]interface{})

	// logger := config.GetLogger()

	// Default value from YAML config
	defaultBlockConfig := &BlockImageResizeConfig{}
	helpers.MapToYAMLStruct(block.GetConfigSection(), defaultBlockConfig)

	// User defined values from data
	userBlockConfig := &BlockImageResizeConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)

	// Merge the default and user defined maps to BlockConfig
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	// var imageBytes []byte
	// imageBytesString, err := helpers.GetValue[string](_data, "image")
	// if err == nil {
	// 	imageBytes = []byte(imageBytesString)
	// } else {
	// 	imageBytes, err = helpers.GetValue[[]byte](_data, "image")
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }
	imageBytes, err := helpers.GetValue[[]byte](_data, "image")
	if err != nil {
		return nil, false, err
	}

	imgBuf := bytes.NewBuffer(imageBytes)
	img, format, err := image.Decode(imgBuf)
	if err != nil {
		return nil, false, err
	}
	config.GetLogger().Debugf("Image format: %s", format)

	resizedImage := imaging.Resize(img, blockConfig.Width, blockConfig.Height, imaging.Lanczos)

	err = png.Encode(output, resizedImage)
	if err != nil {
		config.GetLogger().Fatalf("Failed to encode PNG image: %v", err)
	}

	return output, false, nil
}

type BlockImageResizeConfig struct {
	Width           int  `yaml:"width" json:"width"`
	Height          int  `yaml:"height" json:"height"`
	KeepAspectRatio bool `yaml:"keep_aspect_ratio" json:"keep_aspect_ratio"`
}

type BlockImageResize struct {
	BlockParent
	Config *BlockImageResizeConfig
}

func NewBlockImageResize() *BlockImageResize {
	_config := config.GetConfig()

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

	block.SetConfigSection(_config.Blocks[block.GetId()].Config)
	block.SetProcessor(NewProcessorImageResize())

	return block
}
