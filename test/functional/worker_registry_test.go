package functional_test

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
)

func (suite *FunctionalTestSuite) TestGetWorkerPipelines() {
	// Given
	workerRegistry := registries.GetWorkerRegistry()

	pipelineSlug := "test-pipeline-slug"
	blockId := "test-block-id"

	worker, workerEntry, err := factories.NewWorkerServer(
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
			map[string]interfaces.Block{},
			uuid.New(),
			uuid.New(),
		),
	)
	suite.Nil(err)
	suite.NotNil(worker)
	suite.NotNil(workerEntry)

	// When
	pipelines, err := workerRegistry.GetWorkerPipelines(
		workerEntry,
	)

	// Then
	suite.Nil(err)
	suite.NotEmpty(pipelines)
	suite.Equal(len(pipelines), 1)
	suite.Equal(
		pipelines[pipelineSlug].(map[string]interface{})["slug"],
		pipelineSlug,
	)
}

func (suite *FunctionalTestSuite) TestGetWorkerBlocks() {
	// Given
	workerRegistry := registries.GetWorkerRegistry()

	pipelineSlug := "test-pipeline-slug"
	blockId := "http_request"
	block := blocks.NewBlockHTTP()

	blockAvailability := []bool{true, false}
	for _, available := range blockAvailability {
		block.SetAvailable(available)

		worker, workerEntry, err := factories.NewWorkerServer(
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
		suite.NotNil(worker)
		suite.NotNil(workerEntry)

		// When
		blocks, err := workerRegistry.GetWorkerBlocks(
			workerEntry,
		)

		// Then
		suite.Nil(err)
		suite.NotEmpty(blocks)
		suite.Equal(len(blocks), 1)

		fetchedBlock := blocks[blockId].(map[string]interface{})
		suite.Equal(fetchedBlock["id"], blockId)
		suite.Equal(fetchedBlock["available"].(bool), available)
	}
}

func (suite *FunctionalTestSuite) TestWorkerRegistryGetValidWorkers() {
	// Given
	workerRegistry := registries.GetWorkerRegistry()

	pipelineSlug := "test-pipeline-slug"
	blockId := "test-block-id"
	block := blocks.NewBlockHTTP()
	block.SetAvailable(true)

	worker, workerEntry, err := factories.NewWorkerServer(
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
	suite.NotNil(worker)
	suite.NotNil(workerEntry)

	discoveredEntries := []*zeroconf.ServiceEntry{
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "localhost",
			},
			HostName: "localhost",
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.1")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     8080,
			Text: []string{
				"version=0.1",
				"load=0.00",
				"available=false",
				fmt.Sprintf(
					"blocks=block_http,%s,block_gpu_image_resize",
					blockId,
				),
			},
		},
	}
	discoveredWorkers := []interfaces.Worker{
		dataclasses.NewWorker(discoveredEntries[0]),
		workerEntry,
	}
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

	// When
	workers := workerRegistry.GetValidWorkers(pipelineSlug, blockId)

	// Then
	suite.NotEmpty(workers)
	suite.Equal(len(workers), 1)
	suite.Equal(
		workers[discoveredWorkers[1].GetId()],
		discoveredWorkers[1],
	)
}

func (suite *FunctionalTestSuite) TestWorkerRegistryResumeProcessing() {
	// This is updated copy/paste of Unit TestSuite TestWorkerRegistryResumeProcessing

	// Given
	workerRegistry := registries.GetWorkerRegistry()

	pipelineSlug := "test-pipeline-slug"
	blockId := "test-block-id"
	block := blocks.NewBlockHTTP()
	block.SetAvailable(true)
	processingId := uuid.New()

	worker, workerEntry, err := factories.NewWorkerServer(
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
	suite.NotNil(worker)
	suite.NotNil(workerEntry)

	discoveredEntries := []*zeroconf.ServiceEntry{
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "localhost",
			},
			HostName: "localhost",
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.1")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     8080,
			Text: []string{
				"version=0.1",
				"load=0.00",
				"available=false",
				fmt.Sprintf(
					"blocks=block_http,%s,block_gpu_image_resize",
					blockId,
				),
			},
		},
	}
	discoveredWorkers := []interfaces.Worker{
		dataclasses.NewWorker(discoveredEntries[0]),
		workerEntry,
	}
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

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

func (suite *FunctionalTestSuite) TestWorkerShutdownCorrect() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()
	httpClient := &http.Client{}

	server1, _, err := factories.NewWorkerServerWithHandlers(ctx, true)

	// When
	suite.Nil(err)
	response, err := httpClient.Get(
		fmt.Sprintf(
			"%s/health",
			server1.GetAPIAddress(),
		),
	)

	// Then
	suite.Nil(err)
	suite.Equal(http.StatusOK, response.StatusCode)
}

func (suite *FunctionalTestSuite) TestTwoWorkersDifferentRegistries() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	// When
	server1, worker1, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)
	server2, worker2, err := factories.NewWorkerServerWithHandlers(ctx, false)
	suite.Nil(err)

	// Then
	suite.False(server1.GetWorkerRegistry() == server2.GetWorkerRegistry())
	suite.False(server1.GetPipelineRegistry() == server2.GetPipelineRegistry())
	suite.False(server1.GetBlockRegistry() == server2.GetBlockRegistry())
	suite.False(server1.GetProcessingRegistry() == server2.GetProcessingRegistry())

	server1.GetWorkerRegistry().Add(worker1)
	server1.GetWorkerRegistry().Add(worker2)

	suite.NotEqual(
		server1.GetWorkerRegistry().GetAll(),
		server2.GetWorkerRegistry().GetAll(),
	)
}

func (suite *FunctionalTestSuite) TestTwoWorkersWorkersDiscovery() {
	// Given
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	testPipelineSlug, testBlockId := "test-two-http-blocks", "http_request"
	server1, worker1, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)
	server2, worker2, err := factories.NewWorkerServerWithHandlers(ctx, true)
	suite.Nil(err)

	workerRegistry1 := server1.GetWorkerRegistry()
	workerRegistry2 := server2.GetWorkerRegistry()

	// When
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	// Then
	availableWorkers1 := workerRegistry1.GetAvailableWorkers()
	availableWorkers2 := workerRegistry2.GetAvailableWorkers()
	suite.Equal(len(availableWorkers1), 1)
	suite.Equal(len(availableWorkers2), 1)
	suite.Equal(availableWorkers1[worker2.GetId()], worker2)
	suite.Equal(availableWorkers2[worker1.GetId()], worker1)

	workersWithBlocks1 := workerRegistry1.GetWorkersWithBlocksDetected(availableWorkers1, testBlockId)
	workersWithBlocks2 := workerRegistry2.GetWorkersWithBlocksDetected(availableWorkers2, testBlockId)
	suite.Equal(len(workersWithBlocks1), 1)
	suite.Equal(len(workersWithBlocks2), 1)
	suite.Equal(workersWithBlocks1[worker2.GetId()], worker2)
	suite.Equal(workersWithBlocks2[worker1.GetId()], worker1)

	validWorkers1 := workerRegistry1.GetValidWorkers(testPipelineSlug, testBlockId)
	validWorkers2 := workerRegistry2.GetValidWorkers(testPipelineSlug, testBlockId)
	suite.Equal(len(validWorkers1), 1)
	suite.Equal(len(validWorkers2), 1)
	suite.Equal(validWorkers1[worker2.GetId()], worker2)
	suite.Equal(validWorkers2[worker1.GetId()], worker1)
}
