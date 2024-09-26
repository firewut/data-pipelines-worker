package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/helpers"
)

func (suite *UnitTestSuite) TestGetValue() {
	_data := map[string]interface{}{
		"string":  "value",
		"boolean": true,
		"number":  1,
	}

	valueString, err := helpers.GetValue[string](_data, "string")
	suite.Nil(err)
	suite.Equal("value", valueString)

	valueBool, err := helpers.GetValue[bool](_data, "boolean")
	suite.Nil(err)
	suite.Equal(true, valueBool)

	valueNumber, err := helpers.GetValue[int](_data, "number")
	suite.Nil(err)
	suite.Equal(1, valueNumber)
}

func (suite *UnitTestSuite) TestMapToYAMLStruct() {
	block := blocks.NewBlockImageResize()

	defaultBlockConfig := &blocks.BlockImageResizeConfig{}
	helpers.MapToYAMLStruct(block.GetConfigSection(), defaultBlockConfig)

	suite.Equal(100, defaultBlockConfig.Width)
}

func (suite *UnitTestSuite) TestMapToJSONStruct() {
	userBlockConfig := &blocks.BlockImageResizeConfig{}
	helpers.MapToJSONStruct(map[string]interface{}{"width": 66}, userBlockConfig)

	suite.Equal(66, userBlockConfig.Width)
}

func (suite *UnitTestSuite) TestMergeStructs() {
	defaultBlockConfig := &blocks.BlockImageResizeConfig{Width: 100}
	userBlockConfig := &blocks.BlockImageResizeConfig{Width: 66}
	blockConfig := &blocks.BlockImageResizeConfig{}

	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	suite.Equal(66, blockConfig.Width)
}

func (suite *UnitTestSuite) TestGetListAsQuotedString() {
	list := []string{"a", "b", "c"}
	quotedList := helpers.GetListAsQuotedString(list)

	suite.Equal(`"a", "b", "c"`, quotedList)
}
