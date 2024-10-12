package blocks

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
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
) (*bytes.Buffer, bool, bool, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockImageAddTextConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockImageAddText).GetBlockConfig(_config)
	userBlockConfig := &BlockImageAddTextConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	text, err := helpers.GetValue[string](_data, "text")
	if err != nil {
		return nil, false, false, err
	}

	imageBytes, err := helpers.GetValue[[]byte](_data, "image")
	if err != nil {
		return nil, false, false, err
	}

	imgBuf := bytes.NewBuffer(imageBytes)
	img, format, err := image.Decode(imgBuf)
	if err != nil {
		return nil, false, false, err
	}
	config.GetLogger().Debugf("Image format: %s", format)

	if blockConfig.Font == "" {
		blockConfig.Font = defaultBlockConfig.Font
	}

	fontBytes, err := config.LoadFont(blockConfig.Font)
	if err != nil {
		return nil, false, false, err
	}
	fontParsed, err := truetype.Parse(fontBytes)
	if err != nil {
		return nil, false, false, err
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

	lineHeight := blockConfig.FontSize * 1.2 // Increase the line height for better readability
	maxWidth := float64(dc.Width())          // Use the full width of the context

	var (
		x           float64
		y           float64
		ax          float64
		ay          float64
		width       float64
		lineSpacing float64 = 1.2
		align       gg.Align
	)
	switch blockConfig.TextPosition {
	case "top-left":
		x = lineHeight
		y = lineHeight
		ax = 0
		ay = 0
		width = maxWidth - lineHeight*2
		align = gg.AlignLeft
	case "top-center":
		x = lineHeight
		y = lineHeight
		ax = 0
		ay = 0
		width = maxWidth - lineHeight*2
		align = gg.AlignCenter
	case "top-right":
		x = lineHeight
		y = lineHeight
		ax = 0
		ay = 0
		width = maxWidth - lineHeight*2
		align = gg.AlignRight
	case "center-left":
		x = lineHeight
		y = float64(dc.Height()) / 2
		ax = 0
		ay = 0
		width = maxWidth - lineHeight*2
		align = gg.AlignLeft
	case "center-center":
		x = lineHeight
		y = float64(dc.Height()) / 2
		ax = 0
		ay = 0
		width = maxWidth - lineHeight*2
		align = gg.AlignCenter
	case "center-right":
		x = lineHeight
		y = float64(dc.Height()) / 2
		ax = 0
		ay = 0
		width = maxWidth - lineHeight*2
		align = gg.AlignRight
	case "bottom-left":
		x = lineHeight
		y = float64(dc.Height()) - lineHeight
		ax = 0
		ay = 1
		width = maxWidth - lineHeight*2
		align = gg.AlignLeft
	case "bottom-center":
		x = lineHeight
		y = float64(dc.Height()) - lineHeight
		ax = 0
		ay = 1
		width = maxWidth - lineHeight*2
		align = gg.AlignCenter
	case "bottom-right":
		x = lineHeight
		y = float64(dc.Height()) - lineHeight
		ax = 0
		ay = 1
		width = maxWidth - lineHeight*2
		align = gg.AlignRight
	default:
		x = lineHeight
		y = float64(dc.Height()) / 2
		ax = 0
		ay = 0
		width = maxWidth - lineHeight*2
		align = gg.AlignCenter
	}

	drawTextWithBackground(
		dc, text, x, y, ax, ay, width, lineSpacing, align,
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

	return output, false, false, nil
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

func drawTextWithBackground(dc *gg.Context, s string, x, y, ax, ay float64, width, lineSpacing float64, align gg.Align, blockConfig *BlockImageAddTextConfig) {
	lines := dc.WordWrap(s, width)

	// Sync height formula with MeasureMultilineString
	h := float64(len(lines)) * dc.FontHeight() * lineSpacing
	h -= (lineSpacing - 1) * dc.FontHeight()

	x -= ax * width
	y -= ay * h

	switch align {
	case gg.AlignLeft:
		ax = 0
	case gg.AlignCenter:
		ax = 0.5
		x += width / 2
	case gg.AlignRight:
		ax = 1
		x += width
	}

	ay = 1
	lineHeight := dc.FontHeight() * lineSpacing // Calculate the line height

	// Initialize variables to gather dimensions
	margin := blockConfig.TextBgMargin
	totalHeight := lineHeight*float64(len(lines)) + margin*2

	bgr, bgg, bgb := helpers.HexToRGB(blockConfig.TextBgColor)
	dc.SetRGBA255(int(bgr), int(bgg), int(bgb), int(blockConfig.TextBgAlpha*255))

	if blockConfig.TextBgAllWidth {
		dc.DrawRectangle(0, y-margin/2, float64(dc.Width()), totalHeight)
	} else {
		dc.DrawRectangle(x-margin, y-margin/2, ax+margin, totalHeight)
	}
	dc.Fill()

	// Second pass: Draw the text over the rectangle
	for _, line := range lines {
		// Draw the text anchored
		dc.SetHexColor(blockConfig.FontColor)
		dc.DrawStringAnchored(line, x, y, ax, ay)
		y += lineHeight // Move y down for the next line
	}

}

// TextClip(
// 	txt=params["text"],
// 	fontsize=params["fontsize"],
// 	size=(0.8 * background_clip.size[0], 0),
// 	color=params["text_color"],
// 	method=params["text_method"],
// )
// .set_position("center")
// .set_duration(params["duration"])

// ColorClip(
// 	size=(settings.video_width, int(text_clip.size[1] * 1.4)),
// 	color=params.get(
// 		"text_bg_color", (0, 0, 0)
// 	),  # Default to black if no color is specified
// 	duration=params["duration"],
// )
// .set_opacity(0.8)
// .set_duration(
// 	text_clip.duration,
// )
// .set_position(
// 	"center",
// )

type BlockImageAddText struct {
	generics.ConfigurableBlock[BlockImageAddTextConfig]
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
								"description": "Text Background all width. Always true for now",
								"type": "boolean",
								"default": true,
								"enum": [true]
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
