package blocks

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
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

func (p *ProcessorImageAddText) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorImageAddText) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorImageAddText) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockImageAddTextConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockImageAddText).GetBlockConfig(_config)
	userBlockConfig := &BlockImageAddTextConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	// False is not merging correct over true ( default is false )
	if userBlockConfig.TextBgAllWidth != defaultBlockConfig.TextBgAllWidth {
		blockConfig.TextBgAllWidth = userBlockConfig.TextBgAllWidth
	}

	text, err := helpers.GetValue[string](_data, "text")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	text = strings.Trim(text, " ")

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

	if blockConfig.Font == "" {
		blockConfig.Font = defaultBlockConfig.Font
	}

	fontBytes, err := config.LoadFont(blockConfig.Font)
	if err != nil {
		return nil, false, false, "", -1, err
	}
	fontParsed, err := truetype.Parse(fontBytes)
	if err != nil {
		return nil, false, false, "", -1, err
	}
	fontFace := truetype.NewFace(
		fontParsed,
		&truetype.Options{
			Size:    blockConfig.FontSize,
			DPI:     72,
			Hinting: font.HintingNone,
		},
	)
	defer fontFace.Close()

	dc := gg.NewContextForImage(img)
	dc.SetFontFace(fontFace)

	lineHeight := dc.FontHeight() * 1.2 // Increase the line height for better readability

	var (
		y     float64
		align gg.Align
	)
	switch blockConfig.TextPosition {
	case "top-left":
		y = lineHeight
		align = gg.AlignLeft
	case "top-center":
		y = lineHeight
		align = gg.AlignCenter
	case "top-right":
		y = lineHeight
		align = gg.AlignRight
	case "center-left":
		y = float64(dc.Height()) / 2
		align = gg.AlignLeft
	case "center-center":
		y = float64(dc.Height()) / 2
		align = gg.AlignCenter
	case "center-right":
		y = float64(dc.Height()) / 2
		align = gg.AlignRight
	case "bottom-left":
		y = float64(dc.Height())
		align = gg.AlignLeft
	case "bottom-center":
		y = float64(dc.Height())
		align = gg.AlignCenter
	case "bottom-right":
		y = float64(dc.Height())
		align = gg.AlignRight
	default:
		y = float64(dc.Height()) / 2
		align = gg.AlignCenter
	}

	drawTextWithBackground(
		dc, text, lineHeight, y, align,
		blockConfig,
	)

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

	return output, false, false, "", -1, nil
}

type BlockImageAddTextConfig struct {
	Font           string  `yaml:"font" json:"font"`
	FontSize       float64 `yaml:"font_size" json:"font_size"`
	FontColor      string  `yaml:"font_color" json:"font_color"`
	TextPosition   string  `yaml:"text_position" json:"text_position"`
	TextBgColor    string  `yaml:"text_bg_color" json:"text_bg_color"`
	TextBgAlpha    float64 `yaml:"text_bg_alpha" json:"text_bg_alpha"`
	TextBgMargin   float64 `yaml:"text_bg_margin" json:"text_bg_margin"`
	TextBgAllWidth bool    `yaml:"text_bg_all_width" json:"text_bg_all_width"`
}

func drawTextWithBackground(
	dc *gg.Context,
	s string,
	lineHeight float64,
	y float64,
	align gg.Align,
	blockConfig *BlockImageAddTextConfig,
) {
	margin := blockConfig.TextBgMargin

	// Set background color for the rectangle
	bgr, bgg, bgb := helpers.HexToRGB(blockConfig.TextBgColor)
	dc.SetRGBA255(int(bgr), int(bgg), int(bgb), int(blockConfig.TextBgAlpha*255))

	rectWidth := float64(dc.Width()) * 0.8 // Rectangle width is 80% of canvas width
	rectX := (float64(dc.Width()) - rectWidth) / 2

	// Check if rectangle should span the full width
	if blockConfig.TextBgAllWidth {
		rectX = 0
		rectWidth = float64(dc.Width())
	}

	wrappedText := dc.WordWrap(s, rectWidth-(2*margin))           // Wrap the text within the rectangle
	rectHeight := float64(len(wrappedText))*lineHeight + 2*margin // Calculate rectangle height

	// Edge cases
	switch {
	case y < 0.0:
		// Negative top
		y = 0.0
	case y == float64(dc.Height()):
		// Bottom
		y -= rectHeight
	}

	// Draw the background rectangle
	dc.DrawRectangle(rectX, y, rectWidth, rectHeight)
	dc.Fill()

	// Clip to the rectangle to ensure text doesn't overflow
	dc.Push()
	dc.DrawRectangle(rectX, y, rectWidth, rectHeight)
	dc.Clip()

	// Set the text color
	dc.SetHexColor(blockConfig.FontColor)

	// Adjust textY to properly position the first line of text inside the rectangle
	// Offset the textY by 0.8*lineHeight to account for the baseline shift
	textY := y + margin + 0.8*lineHeight

	// Draw each line of wrapped text with the specified alignment
	for _, line := range wrappedText {
		var textX float64
		textWidth, _ := dc.MeasureString(line)

		// Align text based on the alignment parameter (left, center, right)
		switch align {
		case gg.AlignLeft:
			textX = rectX + margin
		case gg.AlignCenter:
			textX = rectX + (rectWidth-textWidth)/2
		case gg.AlignRight:
			textX = rectX + rectWidth - textWidth - margin
		}

		// Draw the line of text
		dc.DrawString(line, textX, textY)
		// Move textY down for the next line
		textY += lineHeight
	}

	// Restore the context to remove the clipping
	dc.Pop()
}

type BlockImageAddText struct {
	generics.ConfigurableBlock[BlockImageAddTextConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockImageAddText)(nil)

func (b *BlockImageAddText) GetBlockConfig(_config config.Config) *BlockImageAddTextConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockImageAddTextConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockImageAddText() *BlockImageAddText {
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
							"text_bg_color": {
								"description": "Text Background Color",
								"type": "string",
								"format": "color",
								"default": "#000000"
							},
							"text_bg_margin": {
								"description": "Text Background Margin",
								"type": "number",
								"format": "float",
								"default": 10
							},
							"text_bg_alpha": {
								"description": "Text Background Opacity",
								"type": "number",
								"format": "float",
								"default": 0.5
							},
							"text_bg_all_width": {
								"description": "Text Background all width",
								"type": "boolean",
								"default": false
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

	block.SetProcessor(NewProcessorImageAddText())

	return block
}
