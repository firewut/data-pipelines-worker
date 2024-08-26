package unit_test

import (
	"data-pipelines-worker/types"
)

func (suite *UnitTestSuite) TestNewPipelineRegistry() {
	registry := types.NewPipelineRegistry()

	suite.NotNil(registry)
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryRegisterCorrect() {
	registry := types.NewPipelineRegistry()
	pipeline := types.NewPipeline(suite.GetTestPipelineDefinition())

	registry.Register(pipeline)

	suite.NotEmpty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryRegisterErrorInvalidJSON() {
	registry := types.NewPipelineRegistry()

	suite.Panics(func() {
		types.NewPipeline([]byte(`{`))
	})

	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryRegisterErrorMissingRequiredProperty() {
	registry := types.NewPipelineRegistry()
	pipeline := types.NewPipeline([]byte(`{
		"slug": "YT-CHANNEL-video-generation",
		"title": "Youtube Video generation Pipeline"
	}`))

	suite.Panics(func() {
		registry.Register(pipeline)
	})

	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryGet() {
	registry := types.NewPipelineRegistry()
	pipeline := types.NewPipeline(suite.GetTestPipelineDefinition())

	registry.Register(pipeline)
	suite.NotEmpty(registry.GetAll())

	suite.NotEmpty(registry.Get("test"))
}

func (suite *UnitTestSuite) TestPipelineRegistryGetAll() {
	registry := types.NewPipelineRegistry()
	pipeline := types.NewPipeline(suite.GetTestPipelineDefinition())

	registry.Register(pipeline)
	suite.NotEmpty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryDelete() {
	registry := types.NewPipelineRegistry()
	pipeline := types.NewPipeline(suite.GetTestPipelineDefinition())

	registry.Register(pipeline)
	suite.NotEmpty(registry.GetAll())

	registry.Delete("test")
	suite.Empty(registry.GetAll())
}

func (suite *UnitTestSuite) TestPipelineRegistryLoadFromCatalogue() {
	registry := types.NewPipelineRegistry()
	registry.LoadFromCatalogue()

	suite.NotEmpty(registry.GetAll())
}
