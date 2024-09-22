package unit_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/api/schemas"
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
	err := workerRegistry.Shutdown(
		suite.GetShutDownContext(
			time.Second,
		),
	)
	suite.Nil(err)
}

func (suite *UnitTestSuite) TestGetWorkerRegistry() {
	// Given
	cases := [][]bool{
		{false, false, true}, // Expect same instance
		{true, false, false}, // Expect the same instance as the one created
		{false, true, false}, // Expect different instances
		{true, true, false},  // Expect different instances
	}

	// When
	for _, c := range cases {
		workerRegistry1 := registries.GetWorkerRegistry(c[0])
		workerRegistry2 := registries.GetWorkerRegistry(c[1])

		// Then
		suite.Equal(workerRegistry1 == workerRegistry2, c[2])
	}
}

func (suite *UnitTestSuite) TestWorkerRegistryQueryWorkerAPI() {
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
		0,
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
			uuid.New(),
			uuid.New(),
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
		"GET",
		&pipelines,
	)
	suite.Nil(err)
	suite.NotEmpty(pipelines)

	err = workerRegistry.QueryWorkerAPI(
		workerEntry,
		"blocks",
		"GET",
		&blocks,
	)
	suite.Nil(err)
	suite.NotEmpty(blocks)
}

func (suite *UnitTestSuite) TestWorkerRegistryGetValidWorkers() {
	// Given
	workerRegistry := registries.GetWorkerRegistry()

	pipelineSlug := "test-pipeline-slug"
	blockId := "test-block-id"
	block := blocks.NewBlockHTTP()
	block.SetAvailable(true)

	workerServer, workerEntry, err := factories.NewWorkerServer(
		suite.GetMockHTTPServer,
		http.StatusOK,
		true,
		[]string{blockId},
		0,
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
			uuid.New(),
			uuid.New(),
		),
	)
	suite.Nil(err)
	suite.NotNil(workerServer)
	suite.NotNil(workerEntry)

	workerRegistry.Add(workerEntry)

	// When
	validWorkers := workerRegistry.GetValidWorkers(pipelineSlug, blockId)

	// Then
	suite.NotEmpty(validWorkers)
	suite.Equal(1, len(validWorkers))
	suite.Equal(workerEntry, validWorkers[workerEntry.GetId()])
}

func (suite *UnitTestSuite) TestWorkerRegistryResumeProcessing() {
	// Given
	workerRegistry := registries.GetWorkerRegistry()

	pipelineSlug := "test-pipeline-slug"
	blockId := "test-block-id"
	block := blocks.NewBlockHTTP()
	block.SetAvailable(true)
	processingId := uuid.New()

	workerServer, workerEntry, err := factories.NewWorkerServer(
		suite.GetMockHTTPServer,
		http.StatusOK,
		true,
		[]string{blockId},
		0,
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
			uuid.New(),
			processingId,
		),
	)
	suite.Nil(err)
	suite.NotNil(workerServer)
	suite.NotNil(workerEntry)

	workerRegistry.Add(workerEntry)

	// When
	err = workerRegistry.ResumeProcessing(
		pipelineSlug,
		processingId,
		blockId,
		schemas.PipelineStartInputSchema{},
	)

	// Then
	suite.Nil(err)
}
