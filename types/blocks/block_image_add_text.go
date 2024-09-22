package blocks

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"log"

	"github.com/fogleman/gg"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorImageAddText struct {
	BlockDetectorParent
}

func NewDetectorImageAddText(
	detectorConfig config.BlockConfigDetector,
) *DetectorImageAddText {
	return &DetectorImageAddText{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorImageAddText) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorImageAddText struct {
}

func NewProcessorImageAddText() *ProcessorImageAddText {
	return &ProcessorImageAddText{}
}

func (p *ProcessorImageAddText) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, error) {
	var output *bytes.Buffer = &bytes.Buffer{}

	logger := config.GetLogger()
	logger.Debugf("Starting HTTP request for block %s", data.GetSlug())

	blockConfig := &BlockImageAddTextConfig{}
	helpers.MapToYAMLStruct(block.GetConfigSection(), blockConfig)

	_data := data.GetInputData().(map[string]interface{})

	text, err := helpers.GetValue[string](_data, "text")
	if err != nil {
		return nil, err
	}

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

	dc := gg.NewContextForImage(img)

	// Set the font size and load a font (substitute with your own font file)
	if err := dc.LoadFontFace(blockConfig.Font, blockConfig.FontSize); err != nil {
		log.Fatalf("Failed to load font: %v", err)
	}

	dc.SetRGB(0, 0, 0) // White color
	// Draw the text at a specific location
	dc.DrawString(text, 50, 100) // X: 50, Y: 100

	// Convert the image to RGBA to ensure alpha channel is present
	rgbaImage := image.NewRGBA(dc.Image().Bounds())
	for y := 0; y < rgbaImage.Bounds().Dy(); y++ {
		for x := 0; x < rgbaImage.Bounds().Dx(); x++ {
			rgbaImage.Set(x, y, dc.Image().At(x, y))
		}
	}

	err = png.Encode(output, rgbaImage)
	if err != nil {
		log.Fatalf("Failed to encode PNG image: %v", err)
	}

	return output, nil
}

type BlockImageAddTextConfig struct {
	Font     string  `yaml:"font"`
	FontSize float64 `yaml:"font_size"`
}

type BlockImageAddText struct {
	BlockParent
	Config *BlockImageAddTextConfig
}

func NewBlockImageAddText() *BlockImageAddText {
	_config := config.GetConfig()

	block := &BlockImageAddText{
		BlockParent: BlockParent{
			Id:          "image_add_text",
			Name:        "Image Add Text",
			Description: "Add text to Image",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"text": {
								"description": "Text to add to the image",
								"type": "string",
								"minLength": 1
							},
							"image": {
								"description": "Image to add text to",
								"type": ["string", "object"]
							}
						},
						"required": ["text", "image"]
					},
					"output": {
						"description": "Image with text added",
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
	block.SetProcessor(NewProcessorImageAddText())

	return block
}
