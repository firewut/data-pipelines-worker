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

type DetectorBlockImageBlur struct {
	BlockDetectorParent
}

func NewDetectorBlockImageBlur(
	detectorConfig config.BlockConfigDetector,
) *DetectorBlockImageBlur {
	return &DetectorBlockImageBlur{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorBlockImageBlur) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorImageBlur struct {
}

func NewProcessorImageBlur() *ProcessorImageBlur {
	return &ProcessorImageBlur{}
}

func (p *ProcessorImageBlur) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, error) {
	var (
		output      *bytes.Buffer         = &bytes.Buffer{}
		blockConfig *BlockImageBlurConfig = &BlockImageBlurConfig{}
	)
	_data := data.GetInputData().(map[string]interface{})

	logger := config.GetLogger()
	logger.Debugf("Starting HTTP request for block %s", data.GetSlug())

	// Default value from YAML config
	defaultBlockConfig := &BlockImageBlurConfig{}
	helpers.MapToYAMLStruct(block.GetConfigSection(), defaultBlockConfig)

	// User defined values from data
	userBlockConfig := &BlockImageBlurConfig{}
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
		return nil, err
	}

	imgBuf := bytes.NewBuffer(imageBytes)
	img, format, err := image.Decode(imgBuf)
	if err != nil {
		return nil, err
	}
	config.GetLogger().Debugf("Image format: %s", format)

	sigma, err := helpers.GetValue[float64](_data, "sigma")
	if err != nil {
		return nil, err
	}

	blurredImage := imaging.Blur(img, sigma)

	err = png.Encode(output, blurredImage)
	if err != nil {
		config.GetLogger().Fatalf("Failed to encode PNG image: %v", err)
	}

	return output, nil
}

type BlockImageBlurConfig struct {
	Sigma float64 `yaml:"sigma" json:"sigma"`
}

type BlockImageBlur struct {
	BlockParent
	Config *BlockImageBlurConfig
}

func NewBlockImageBlur() *BlockImageBlur {
	_config := config.GetConfig()

	block := &BlockImageBlur{
		BlockParent: BlockParent{
			Id:          "image_blur",
			Name:        "Image Blur",
			Description: "Blur Image",
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
							"sigma": {
								"description": "Blur sigma",
								"type": "number",
								"format": "float",
								"default": 1.5
							}
						},
						"required": ["image"]
					},
					"output": {
						"description": "Blurd image",
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
	block.SetProcessor(NewProcessorImageBlur())

	return block
}
