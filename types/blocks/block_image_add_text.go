package blocks

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

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
) (*bytes.Buffer, bool, error) {
	var (
		output      *bytes.Buffer            = &bytes.Buffer{}
		blockConfig *BlockImageAddTextConfig = &BlockImageAddTextConfig{}
	)
	_data := data.GetInputData().(map[string]interface{})

	// logger := config.GetLogger()

	// Default value from YAML config
	defaultBlockConfig := &BlockImageAddTextConfig{}
	helpers.MapToYAMLStruct(block.GetConfigSection(), defaultBlockConfig)

	// User defined values from data
	userBlockConfig := &BlockImageAddTextConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)

	// Merge the default and user defined maps to BlockConfig
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	text, err := helpers.GetValue[string](_data, "text")
	if err != nil {
		return nil, false, err
	}

	// var imageBytes []byte
	// imageBytesString, err := helpers.GetValue[string](_data, "image")
	// if err == nil {
	// 	imageBytes = []byte(imageBytesString)
	// } else {
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

	if blockConfig.Font == "" {
		blockConfig.Font = defaultBlockConfig.Font
	}

	dc := gg.NewContextForImage(img)
	fontBytes, err := config.LoadFont(blockConfig.Font)
	if err != nil {
		return nil, false, err
	}

	fontParsed, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, false, err
	}

	fontFace, err := opentype.NewFace(fontParsed, &opentype.FaceOptions{
		Size:    blockConfig.FontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, false, err
	}
	defer fontFace.Close()

	dc.SetFontFace(fontFace)

	// Set text Color from config
	dc.SetHexColor(blockConfig.FontColor)

	switch blockConfig.TextPosition {
	case "top-left":
		dc.DrawStringAnchored(text, 0, blockConfig.FontSize, 0, 0)
	case "top-center":
		dc.DrawStringAnchored(text, float64(dc.Width())/2, blockConfig.FontSize, 0.5, 0)
	case "top-right":
		dc.DrawStringAnchored(text, float64(dc.Width()), blockConfig.FontSize, 1, 0)
	case "center-left":
		dc.DrawStringAnchored(text, 0, float64(dc.Height())/2, 0, 0.5)
	case "center-center":
		dc.DrawStringAnchored(text, float64(dc.Width())/2, float64(dc.Height())/2, 0.5, 0.5)
	case "center-right":
		dc.DrawStringAnchored(text, float64(dc.Width()), float64(dc.Height())/2, 1, 0.5)
	case "bottom-left":
		dc.DrawStringAnchored(text, 0, float64(dc.Height())-blockConfig.FontSize, 0, 1)
	case "bottom-center":
		dc.DrawStringAnchored(text, float64(dc.Width())/2, float64(dc.Height())-blockConfig.FontSize, 0.5, 1)
	case "bottom-right":
		dc.DrawStringAnchored(text, float64(dc.Width()), float64(dc.Height())-blockConfig.FontSize, 1, 1)
	default:
		dc.DrawStringAnchored(text, float64(dc.Width())/2, float64(dc.Height())/2, 0.5, 0.5)
	}

	// Convert the image to RGBA to ensure alpha channel is present
	rgbaImage := image.NewRGBA(dc.Image().Bounds())
	for y := 0; y < rgbaImage.Bounds().Dy(); y++ {
		for x := 0; x < rgbaImage.Bounds().Dx(); x++ {
			rgbaImage.Set(x, y, dc.Image().At(x, y))
		}
	}

	err = png.Encode(output, rgbaImage)
	if err != nil {
		config.GetLogger().Fatalf("Failed to encode PNG image: %v", err)
	}

	return output, false, nil
}

type BlockImageAddTextConfig struct {
	Font         string  `yaml:"font" json:"font"`
	FontSize     float64 `yaml:"font_size" json:"font_size"`
	FontColor    string  `yaml:"font_color" json:"font_color"`
	TextPosition string  `yaml:"text_position" json:"text_position"`
}

type BlockImageAddText struct {
	BlockParent
	Config *BlockImageAddTextConfig
}

func NewBlockImageAddText() *BlockImageAddText {
	_config := config.GetConfig()

	fontsEmbedded, err := config.ListFonts()
	if err != nil {
		panic(err)
	}
	listAsEnum := helpers.GetListAsQuotedString(fontsEmbedded)

	block := &BlockImageAddText{
		BlockParent: BlockParent{
			Id:          "image_add_text",
			Name:        "Image Add Text",
			Description: "Add text to Image",
			Version:     "1",
			SchemaString: fmt.Sprintf(`{
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
								"type": "string",
								"format": "file"
							},
							"font_size": {
								"description": "Font size",
								"type": "number",
								"format": "int"
							},
							"font_color": {
								"description": "Text color",
								"type": "string",
								"format": "color"
							},
							"font": {
								"description": "Font name",
								"type": "string",
								"enum": [
									%s
								],
								"default": "Roboto-Regular.ttf"
							},
							"text_position": {
								"description": "Text position",
								"type": "string",
								"enum": [
									"top-left", 
									"top-center",
									"top-right", 
									"center-left",
									"center-center",
									"center-right",
									"bottom-left",
									"bottom-center", 
									"bottom-right"
								],
								"default": "center-center"
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
				listAsEnum,
			),
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
