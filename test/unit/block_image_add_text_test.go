package unit_test

import (
	"bytes"
	"image"
	"image/png"
	"sync"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/validators"
)

func (suite *UnitTestSuite) TestBlockImageAddText() {
	block := blocks.NewBlockImageAddText()

	suite.Equal("image_add_text", block.GetId())
	suite.Equal("Image Add Text", block.GetName())
	suite.Equal("Add text to Image", block.GetDescription())
	suite.NotNil(block.GetSchema())
	suite.NotEmpty(block.GetSchemaString())

	blockConfig := block.GetBlockConfig(suite._config)
	suite.Equal(50.0, blockConfig.FontSize)
}

func (suite *UnitTestSuite) TestBlockImageAddTextValidateSchemaOk() {
	block := blocks.NewBlockImageAddText()

	schemaPtr, schema, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.Nil(err)
	suite.NotNil(schemaPtr)
	suite.NotNil(schema)
}

func (suite *UnitTestSuite) TestBlockImageAddTextValidateSchemaFail() {
	block := blocks.NewBlockImageAddText()

	block.SchemaString = "{invalid schema"

	_, _, err := block.ValidateSchema(validators.JSONSchemaValidator{})
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockImageAddTextProcessIncorrectInput() {
	// Given
	block := blocks.NewBlockImageAddText()
	data := &dataclasses.BlockData{
		Id:   "image_add_text",
		Slug: "image-add-text",
		Input: map[string]interface{}{
			"text":  "test",
			"image": nil,
		},
	}
	data.SetBlock(block)

	// When
	result, stop, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorImageAddText(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err)
}

func (suite *UnitTestSuite) TestBlockImageAddTextProcessSuccess() {
	// Given
	width := 1024
	height := 1792

	texts := []string{
		"On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.",
	}
	fonts := []string{"Roboto-Regular.ttf"}
	fontSizes := []int{50}
	fontColors := []string{"#AA44AA"}
	textPositions := []string{
		"top-left",
		"top-center",
		"top-right",
		"center-left",
		"center-center",
		"center-right",
		"bottom-left",
		"bottom-center",
		"bottom-right",
	}
	textBgColors := []string{"#000000"}
	textBgAlphas := []float64{0.8}
	textBgMargins := []float64{20}
	textBgAllWidths := []bool{true}
	type testCase struct {
		text           string
		font           string
		fontSize       int
		fontColor      string
		textPosition   string
		textBgColor    string
		textBgAlpha    float64
		textBgMargin   float64
		textBgAllWidth bool
	}
	testCases := []testCase{}
	for _, text := range texts {
		for _, fontSize := range fontSizes {
			for _, fontColor := range fontColors {
				for _, textPosition := range textPositions {
					for _, font := range fonts {
						for _, textBgColor := range textBgColors {
							for _, textBgAlpha := range textBgAlphas {
								for _, textBgMargin := range textBgMargins {
									for _, textBgAllWidth := range textBgAllWidths {
										testCases = append(testCases, testCase{
											text:           text,
											font:           font,
											fontSize:       fontSize,
											fontColor:      fontColor,
											textPosition:   textPosition,
											textBgColor:    textBgColor,
											textBgAlpha:    textBgAlpha,
											textBgMargin:   textBgMargin,
											textBgAllWidth: textBgAllWidth,
										})
									}
								}
							}
						}
					}
				}
			}
		}
	}

	images := make([]image.Image, 0)
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	for _, tc := range testCases {
		wg.Add(1)

		go func(tc testCase) {
			defer wg.Done()
			imageBuffer := factories.GetPNGImageBuffer(width, height)

			block := blocks.NewBlockImageAddText()
			data := &dataclasses.BlockData{
				Id:   "image_add_text",
				Slug: "image-add-text",
				Input: map[string]interface{}{
					"text":              tc.text,
					"font":              tc.font,
					"font_size":         tc.fontSize,
					"font_color":        tc.fontColor,
					"image":             imageBuffer.Bytes(),
					"text_position":     tc.textPosition,
					"text_bg_color":     tc.textBgColor,
					"text_bg_alpha":     tc.textBgAlpha,
					"text_bg_margin":    tc.textBgMargin,
					"text_bg_all_width": tc.textBgAllWidth,
				},
			}
			data.SetBlock(block)

			// When
			result, stop, _, err := block.Process(
				suite.GetContextWithcancel(),
				blocks.NewProcessorImageAddText(),
				data,
			)

			// Then
			suite.NotNil(result)
			suite.False(stop)
			suite.Nil(err)

			image, err := png.Decode(bytes.NewReader(result.Bytes()))
			suite.Nil(err)
			suite.NotNil(image)
			suite.Equal(width, image.Bounds().Dx())
			suite.Equal(height, image.Bounds().Dy())

			// // Save image to local png
			// os.MkdirAll("images", os.ModePerm)
			// out, _ := os.Create(fmt.Sprintf("images/%s.png", tc.textPosition))
			// defer out.Close()

			// png.Encode(out, image)

			mu.Lock()
			images = append(images, image)
			mu.Unlock()
		}(tc)

	}

	wg.Wait()

	// Check that all images are different
	for i := 0; i < len(images); i++ {
		for j := i + 1; j < len(images); j++ {
			suite.NotEqual(images[i], images[j])
		}
	}
}
