package unit_test

import (
	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/registries"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grandcat/zeroconf"
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

func (suite *UnitTestSuite) TestGetWorkerRegistryGetValidWorkersExists() {
	// Given
	pipelineSlug := "test-pipeline-slug"
	blockId := "test-block-id"
	block := blocks.NewBlockHTTP()
	block.SetAvailable(true)

	server := suite.GetMockHTTPServer(
		"OK",
		http.StatusOK,
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
	u, err := url.Parse(server.URL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		fmt.Println("Error converting port to integer:", err)
		return
	}

	workerRegistry := registries.GetWorkerRegistry()
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
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "remotehost",
			},
			HostName: u.Hostname(),
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.2")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     port,
			Text: []string{
				"version=0.1",
				"load=0.00",
				"available=true",
				fmt.Sprintf(
					"blocks=block_http,%s,block_gpu_image_resize",
					blockId,
				),
			},
		},
	}
	discoveredWorkers := []interfaces.Worker{
		dataclasses.NewWorker(discoveredEntries[0]),
		dataclasses.NewWorker(discoveredEntries[1]),
	}
	for _, worker := range discoveredWorkers {
		workerRegistry.Add(worker)
	}

	// When
	workers := workerRegistry.GetValidWorkers(
		pipelineSlug,
		blockId,
	)

	// Then
	suite.NotEmpty(workers)
	suite.Equal(len(workers), 1)
	suite.Equal(
		workers[discoveredWorkers[1].GetId()],
		discoveredWorkers[1],
	)
}
