package functional_test

import (
	"fmt"
	"net"
	"net/http"

	"github.com/grandcat/zeroconf"

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
			map[string]interfaces.Block{},
		),
	)
	suite.Nil(err)
	suite.NotNil(workerServer)
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

func (suite *FunctionalTestSuite) TestGetWorkerRegistryGetValidWorkersExists() {
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
