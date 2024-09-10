package unit_test

import (
	"data-pipelines-worker/types/registries"
)

func (suite *UnitTestSuite) TestNewWorkerRegistry() {
	workerRegistry := registries.NewWorkerRegistry()

	registeredWorkers := workerRegistry.GetAll()
	suite.Equal(len(registeredWorkers), 0)
}

func (suite *UnitTestSuite) TestNewWorkerRegistryAddWorker() {
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

func (suite *UnitTestSuite) TestNewWorkerRegistryGetWorker() {
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

func (suite *UnitTestSuite) TestNewWorkerRegistryGetWorkers() {
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

func (suite *UnitTestSuite) TestNewWorkerRegistryRemoveWorker() {
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
