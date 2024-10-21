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

type DetectorImageBlur struct {
	BlockDetectorParent
}

func NewDetectorImageBlur(
	detectorConfig config.BlockConfigDetector,
) *DetectorImageBlur {
	return &DetectorImageBlur{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorImageBlur) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorImageBlur struct {
}

func NewProcessorImageBlur() *ProcessorImageBlur {
	return &ProcessorImageBlur{}
}

func (p *ProcessorImageBlur) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorImageBlur) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorImageBlur) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockImageBlurConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockImageBlur).GetBlockConfig(_config)
	userBlockConfig := &BlockImageBlurConfig{}
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

	blurredImage := imaging.Blur(img, blockConfig.Sigma)

	err = png.Encode(output, blurredImage)
	if err != nil {
		config.GetLogger().Fatalf("Failed to encode PNG image: %v", err)
	}

	return output, false, false, "", -1, nil
}

type BlockImageBlurConfig struct {
	Sigma float64 `yaml:"sigma" json:"sigma"`
}

type BlockImageBlur struct {
	generics.ConfigurableBlock[BlockImageBlurConfig]
	BlockParent
}

var _ interfaces.Block = (*BlockImageBlur)(nil)

func (b *BlockImageBlur) GetBlockConfig(_config config.Config) *BlockImageBlurConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockImageBlurConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockImageBlur() *BlockImageBlur {
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
						"description": "Blurred image",
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

	block.SetProcessor(NewProcessorImageBlur())

	return block
}
