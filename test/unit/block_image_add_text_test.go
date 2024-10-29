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
	suite.NotNil(err, err)
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
	result, stop, _, _, _, err := block.Process(
		suite.GetContextWithcancel(),
		blocks.NewProcessorImageAddText(),
		data,
	)

	// Then
	suite.Empty(result)
	suite.False(stop)
	suite.NotNil(err, err)
}

func (suite *UnitTestSuite) TestBlockImageAddTextProcessSuccess() {
	// Given
	width := 100
	height := 100

	texts := []string{
		" On October 29, 1929, the United States experienced the Wall Street crash, ",
		" marking the beginning of the Great Depression. ",
		" Stocks plummeted, affecting economies worldwide and leading to economic reforms. ",
	}
	fonts := []string{"Roboto-Regular.ttf"}
	fontSizes := []int{5}
	fontColors := []string{"#FFFFFF"}
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
	textBgMargins := []float64{2.5}
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

	// Prepare all combinations of test cases
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

	// Pre-allocate slice for images
	images := make([]image.Image, len(testCases))
	var wg sync.WaitGroup

	// Run each test case in parallel
	for idx, tc := range testCases {
		wg.Add(1)

		go func(tc testCase, idx int) {
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

			// Process the block
			result, stop, _, _, _, err := block.Process(
				suite.GetContextWithcancel(),
				blocks.NewProcessorImageAddText(),
				data,
			)

			// Validate the result
			suite.NotNil(result)
			suite.False(stop)
			suite.Nil(err)

			// os.MkdirAll("images", os.ModePerm)
			// f, _ := os.Create(fmt.Sprintf("images/image_add_text_%d.png", idx))
			// f.Write(result.Bytes())
			// f.Close()

			// Decode the resulting image
			image, err := png.Decode(bytes.NewReader(result.Bytes()))
			suite.Nil(err)
			suite.NotNil(image)
			suite.Equal(width, image.Bounds().Dx())
			suite.Equal(height, image.Bounds().Dy())

			// Save the result in the pre-allocated slice
			images[idx] = image
		}(tc, idx)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check that all images are different
	for i := 0; i < len(images); i++ {
		for j := i + 1; j < len(images); j++ {
			suite.NotEqual(images[i], images[j])
		}
	}
}
