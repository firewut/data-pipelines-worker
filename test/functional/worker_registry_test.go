package functional_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/interfaces"
)

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

func (suite *FunctionalTestSuite) TestTwoWorkersAPIDiscoveryCommunication() {
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

func (suite *FunctionalTestSuite) TestTwoWorkersPipelineProcessingRequiredBlocksDisabledEverywhere() {
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
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	blockRegistry1 := server1.GetBlockRegistry()
	blockRegistry2 := server2.GetBlockRegistry()
	blockRegistry1.GetAvailableBlocks()[testBlockId].SetAvailable(false)
	blockRegistry2.GetAvailableBlocks()[testBlockId].SetAvailable(false)

	suite.NotContains(blockRegistry1.GetAvailableBlocks(), testBlockId) // Ensure block is deleted
	suite.NotContains(blockRegistry2.GetAvailableBlocks(), testBlockId)

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: testPipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: "test-block-first-slug",
			Input: map[string]interface{}{
				"url": firstBlockInput,
			},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server1,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	processingRegistry1 := server1.GetProcessingRegistry()

	failedProcessing1 := <-processingRegistry1.GetProcessingCompletedChannel()
	suite.Equal(failedProcessing1.GetId(), processingResponse.ProcessingID)

	processing1 := processingRegistry1.Get(processingResponse.ProcessingID.String())
	suite.NotNil(processing1)

	suite.NotNil(processing1.GetError())
	suite.Equal(interfaces.ProcessingStatusFailed, processing1.GetStatus())
}

func (suite *FunctionalTestSuite) TestTwoWorkersPipelineProcessingRequiredBlocksDisabledFirst() {
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
	workerRegistry1.Add(worker2)
	workerRegistry2.Add(worker1)

	blockRegistry1 := server1.GetBlockRegistry()
	blockRegistry2 := server2.GetBlockRegistry()
	blockRegistry1.GetAvailableBlocks()[testBlockId].SetAvailable(false)
	blockRegistry2.GetAvailableBlocks()[testBlockId].SetAvailable(true)

	suite.NotContains(blockRegistry1.GetAvailableBlocks(), testBlockId) // Ensure block is deleted
	suite.Contains(blockRegistry2.GetAvailableBlocks(), testBlockId)

	mockedSecondBlockResponse := fmt.Sprintf("Hello, world! Mocked value is %s", uuid.NewString())
	secondBlockInput := suite.GetMockHTTPServerURL(mockedSecondBlockResponse, http.StatusOK, 0)
	firstBlockInput := suite.GetMockHTTPServerURL(secondBlockInput, http.StatusOK, 0)

	inputData := schemas.PipelineStartInputSchema{
		Pipeline: schemas.PipelineInputSchema{
			Slug: testPipelineSlug,
		},
		Block: schemas.BlockInputSchema{
			Slug: "test-block-first-slug",
			Input: map[string]interface{}{
				"url": firstBlockInput,
			},
		},
	}

	// When
	processingResponse, statusCode, errorResponse, err := suite.SendProcessingStartRequest(
		server1,
		inputData,
		nil,
	)

	// Then
	suite.Empty(errorResponse)
	suite.Nil(err, errorResponse)
	suite.Equal(http.StatusOK, statusCode, errorResponse)
	suite.NotNil(processingResponse.ProcessingID)

	processingRegistry1 := server1.GetProcessingRegistry()
	processingRegistry2 := server2.GetProcessingRegistry()

	transferredProcessing1 := <-processingRegistry1.GetProcessingCompletedChannel()
	suite.Equal(transferredProcessing1.GetId(), processingResponse.ProcessingID)
	suite.Equal(interfaces.ProcessingStatusTransferred, transferredProcessing1.GetStatus())
	suite.Nil(transferredProcessing1.GetError())
	processingRegistry1Processing := processingRegistry1.Get(processingResponse.ProcessingID.String())
	suite.NotEmpty(processingRegistry1Processing)
	suite.Equal(transferredProcessing1, processingRegistry1Processing)

	completedProcessing21 := <-processingRegistry2.GetProcessingCompletedChannel()
	suite.Equal(completedProcessing21.GetId(), processingResponse.ProcessingID)
	processingRegistry2Processing1 := processingRegistry2.Get(processingResponse.ProcessingID.String())
	suite.NotEmpty(processingRegistry2Processing1)
	suite.Equal(completedProcessing21, processingRegistry2Processing1)
	suite.Equal(secondBlockInput, completedProcessing21.GetOutput().GetValue().String())

	completedProcessing22 := <-processingRegistry2.GetProcessingCompletedChannel()
	suite.Equal(completedProcessing22.GetId(), processingResponse.ProcessingID)
	processingRegistry2Processing2 := processingRegistry2.Get(processingResponse.ProcessingID.String())
	suite.NotEmpty(processingRegistry2Processing2)
	suite.Equal(completedProcessing22, processingRegistry2Processing2)
	suite.Equal(mockedSecondBlockResponse, completedProcessing22.GetOutput().GetValue().String())
}

// func (suite *FunctionalTestSuite) TestTwoWorkersPipelineProcessing() {
// 	// Given
// 	ctx, shutdown := context.WithCancel(context.Background())
// 	defer shutdown()

// 	testPipelineSlug := "test-three-blocks"
// 	numWorkers := 3

// 	servers := make([]*api.Server, 0)
// 	workers := make([]interfaces.Worker, 0)
// 	workerRegistries := make([]interfaces.WorkerRegistry, 0)
// 	blockRegistries := make([]interfaces.BlockRegistry, 0)
// 	for i := 0; i < numWorkers; i++ {
// 		server, worker, err := factories.NewWorkerServerWithHandlers(ctx, true)
// 		suite.Nil(err)
// 		servers = append(servers, server)
// 		workers = append(workers, worker)
// 		workerRegistries = append(workerRegistries, server.GetWorkerRegistry())
// 		blockRegistries = append(blockRegistries, server.GetBlockRegistry())
// 	}

// 	for i := 0; i < numWorkers; i++ {
// 		for j := 0; j < numWorkers; j++ {
// 			if i != j {
// 				workerRegistries[i].Add(workers[j])
// 			}
// 		}
// 	}

// 	cases := []struct {
// 		worker1block1Available bool
// 		worker1block2Available bool
// 		worker2block1Available bool
// 		worker2block2Available bool
// 		worker1worker2Transfer bool
// 		worker1block1Error     bool
// 		worker1block2Error     bool
// 		worker2block1Error     bool
// 		worker2block2Error     bool
// 	}{
// 		// Worker1 has all blocks available, Worker2 has all blocks available, no transfer, no errors
// 		{
// 			true, true, true, true, false, false, false, false, false,
// 		},
// 		{
// 			true, true, true, true, true, false, false, false, false,
// 		},
// 		{
// 			true, true, true, true, true, true, false, false, false,
// 		},
// 		{
// 			true, true, true, true, false, false, false, false, false,
// 		},
// 	}

// 	for _, c := range cases {
// 		blockRegistry1.GetAvailableBlocks()[testBlockId].SetAvailable(c.worker1block1Available)
// 	}

// }
