package registries

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"data-pipelines-worker/types/interfaces"

	"github.com/google/uuid"
)

var (
	onceWorkerRegistry     sync.Once
	workerRegistryInstance *WorkerRegistry
)

func GetWorkerRegistry() *WorkerRegistry {
	onceWorkerRegistry.Do(func() {
		workerRegistryInstance = NewWorkerRegistry()
	})

	return workerRegistryInstance
}

type WorkerRegistry struct {
	sync.Mutex

	Workers map[string]interfaces.Worker
}

func NewWorkerRegistry() *WorkerRegistry {
	registry := &WorkerRegistry{
		Workers: make(map[string]interfaces.Worker),
	}

	return registry
}

func (wr *WorkerRegistry) Add(worker interfaces.Worker) {
	wr.Lock()
	defer wr.Unlock()

	wr.Workers[worker.GetId()] = worker
}

func (wr *WorkerRegistry) Get(id string) interfaces.Worker {
	wr.Lock()
	defer wr.Unlock()

	return wr.Workers[id]
}

func (wr *WorkerRegistry) GetAll() map[string]interfaces.Worker {
	wr.Lock()
	defer wr.Unlock()

	return wr.Workers
}

func (wr *WorkerRegistry) Delete(id string) {
	wr.Lock()
	defer wr.Unlock()

	delete(wr.Workers, id)
}

func (wr *WorkerRegistry) DeleteAll() {
	wr.Lock()
	defer wr.Unlock()

	for id := range wr.Workers {
		delete(wr.Workers, id)
	}
}

func (wr *WorkerRegistry) Shutdown() {
}

func (wr *WorkerRegistry) ResumeProcessing(
	pipelineSlug string,
	processingId uuid.UUID,
	blockId string,
) error {
	// validWorkers := wr.GetValidWorkers()

	return nil
}

func (wr *WorkerRegistry) GetValidWorkers(
	pipelineSlug string,
	blockId string,
) map[string]interfaces.Worker {
	wr.Lock()
	defer wr.Unlock()

	validWorkers := make(map[string]interfaces.Worker)
	for id, worker := range wr.Workers {
		if worker.GetStatus().GetAvailable() {
			validWorkers[id] = worker
		}
	}

	for id, worker := range validWorkers {
		workerBlocks := worker.GetStatus().GetBlocks()
		deleteWorker := true
		for _, workerBlockId := range workerBlocks {
			if workerBlockId == blockId {
				deleteWorker = false
			}
		}
		if deleteWorker {
			delete(validWorkers, id)
		}
	}

	for _, worker := range validWorkers {
		// Check Pipelines
		workerPipelines, err := wr.GetWorkerPipelines(worker)
		if err != nil {
			delete(validWorkers, worker.GetId())
			continue
		}
		for workerPipelineSlug, _ := range workerPipelines {
			deleteWorker := true
			if workerPipelineSlug == pipelineSlug {
				deleteWorker = false
			}
			if deleteWorker {
				delete(validWorkers, worker.GetId())
			}
		}

		// Check Blocks
		workerBlocks, err := wr.GetWorkerBlocks(worker)
		if err != nil {
			delete(validWorkers, worker.GetId())
			continue
		}
		deleteWorker := true
		for workerBlockId, workerBlock := range workerBlocks {
			if workerBlockId != blockId {
				continue
			}
			if workerBlockMap, ok := workerBlock.(map[string]interface{}); ok {
				if _, ok := workerBlockMap["available"]; !ok {
					continue
				}
				if workerBlockMap["available"].(bool) {
					deleteWorker = false
				}

			}
		}
		if deleteWorker {
			delete(validWorkers, worker.GetId())
			continue
		}
	}

	return validWorkers
}

func (wr *WorkerRegistry) QueryWorkerAPI(
	worker interfaces.Worker,
	path string,
	result interface{},
) error {
	request, err := http.Get(
		fmt.Sprintf(
			"%s/%s",
			worker.GetAPIEndpoint(),
			path,
		),
	)
	if err != nil {
		return err
	}
	defer request.Body.Close()

	err = json.NewDecoder(request.Body).Decode(result)
	if err != nil {
		return err
	}

	return nil
}

func (wr *WorkerRegistry) GetWorkerPipelines(worker interfaces.Worker) (
	map[string]interface{},
	error,
) {
	pipelines := make(map[string]interface{})
	err := wr.QueryWorkerAPI(worker, "pipelines", &pipelines)
	return pipelines, err
}

func (wr *WorkerRegistry) GetWorkerBlocks(worker interfaces.Worker) (
	map[string]interface{},
	error,
) {
	blocks := make(map[string]interface{})
	err := wr.QueryWorkerAPI(worker, "blocks", &blocks)
	return blocks, err
}
