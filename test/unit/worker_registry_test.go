package unit_test

import (
	"fmt"
	"net/http"

	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

func (suite *UnitTestSuite) TestNewWorkerRegistry() {
	workerRegistry := registries.NewWorkerRegistry()

	registeredWorkers := workerRegistry.GetAll()
	suite.Equal(len(registeredWorkers), 0)
}

func (suite *UnitTestSuite) TestNewWorkerRegistryAdd() {
	workerRegistry := registries.NewWorkerRegistry()

	discoveredWorkers := suite.GetDiscoveredWorkers()
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

	suite.Equal(
		len(discoveredWorkers),
		len(workerRegistry.GetAll()),
	)
}

func (suite *UnitTestSuite) TestNewWorkerRegistryGet() {
	workerRegistry := registries.NewWorkerRegistry()

	discoveredWorkers := suite.GetDiscoveredWorkers()
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

	for _, worker := range discoveredWorkers {
		suite.Equal(
			worker,
			workerRegistry.Get(worker.GetId()),
		)
	}
}

func (suite *UnitTestSuite) TestNewWorkerRegistryGetAll() {
	workerRegistry := registries.NewWorkerRegistry()

	discoveredWorkers := suite.GetDiscoveredWorkers()
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

	suite.Equal(
		len(discoveredWorkers),
		len(workerRegistry.GetAll()),
	)
}

func (suite *UnitTestSuite) TestNewWorkerRegistryRemove() {
	workerRegistry := registries.NewWorkerRegistry()

	discoveredWorkers := suite.GetDiscoveredWorkers()
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

	for _, worker := range discoveredWorkers {
		workerRegistry.Delete(worker.GetId())
	}

	suite.Equal(len(workerRegistry.GetAll()), 0)
}

func (suite *UnitTestSuite) TestNewWorkerRegistryRemoveAll() {
	workerRegistry := registries.NewWorkerRegistry()

	discoveredWorkers := suite.GetDiscoveredWorkers()
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

	workerRegistry.DeleteAll()

	suite.Equal(len(workerRegistry.GetAll()), 0)
}

func (suite *UnitTestSuite) TestNewWorkerRegistryShutdown() {
	workerRegistry := registries.NewWorkerRegistry()
	workerRegistry.Shutdown()
}

func (suite *UnitTestSuite) TestGetWorkerRegistry() {
	workerRegistry := registries.GetWorkerRegistry()
	suite.NotEmpty(workerRegistry)

	discoveredWorkers := suite.GetDiscoveredWorkers()
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

	workerRegistry2 := registries.GetWorkerRegistry()
	suite.Equal(
		len(workerRegistry.GetAll()),
		len(workerRegistry2.GetAll()),
	)
}

func (suite *UnitTestSuite) TestQueryWorkerAPI() {
	// Given
	workerRegistry := registries.GetWorkerRegistry()

	pipelineSlug := "test-pipeline-slug"
	blockId := "test-block-id"
	block := blocks.NewBlockHTTP()

	workerServer, workerEntry, err := factories.NewWorkerServer(
		suite.GetMockHTTPServer,
		http.StatusOK,
		true,
		[]string{blockId},
		suite.GetMockServerHandlersResponse(
			map[string]interfaces.Pipeline{
				pipelineSlug: suite.GetTestPipeline(
					fmt.Sprintf(`{
							"slug": "%s",
							"title": "Test Pipeline",
							"description": "Test Pipeline Description",
							"blocks": [
								{
									"id": "%s",
									"slug": "test-block-slug",
									"description": "Do something"
								}
							]
						}`,
						pipelineSlug,
						blockId,
					),
				),
			},
			map[string]interfaces.Block{blockId: block},
		),
	)
	suite.Nil(err)
	suite.NotNil(workerServer)
	suite.NotNil(workerEntry)

	// When
	pipelines := make(map[string]interface{})
	blocks := make(map[string]interface{})

	err = workerRegistry.QueryWorkerAPI(
		workerEntry,
		"pipelines",
		&pipelines,
	)
	suite.Nil(err)
	suite.NotEmpty(pipelines)

	err = workerRegistry.QueryWorkerAPI(
		workerEntry,
		"blocks",
		&blocks,
	)
	suite.Nil(err)
	suite.NotEmpty(blocks)
}
