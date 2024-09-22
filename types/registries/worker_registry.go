package registries

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/interfaces"
)

var (
	onceWorkerRegistry     sync.Once
	workerRegistryInstance *WorkerRegistry
)

func GetWorkerRegistry(forceNewInstance ...bool) *WorkerRegistry {
	if len(forceNewInstance) > 0 && forceNewInstance[0] {
		newInstance := NewWorkerRegistry()
		workerRegistryInstance = newInstance
		onceWorkerRegistry = sync.Once{}
		return newInstance
	}

	onceWorkerRegistry.Do(func() {
		workerRegistryInstance = NewWorkerRegistry()
	})

	return workerRegistryInstance
}

// WorkerRegistry is a registry for discovered Workers
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

func (wr *WorkerRegistry) Shutdown(context context.Context) error {
	wr.Lock()
	defer wr.Unlock()

	return nil
}

func (wr *WorkerRegistry) ResumeProcessing(
	pipelineSlug string,
	processingId uuid.UUID,
	blockId string,
	inputData schemas.PipelineStartInputSchema,
) error {
	validWorkers := wr.GetValidWorkers(pipelineSlug, blockId)
	if len(validWorkers) == 0 {
		return fmt.Errorf(
			"no valid workers found for pipeline %s and block %s",
			pipelineSlug,
			blockId,
		)
	}

	for _, validWorker := range validWorkers {
		err := wr.ResumeProcessingAtWorker(
			validWorker,
			pipelineSlug,
			processingId,
			inputData,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (wr *WorkerRegistry) GetAvailableWorkers() map[string]interfaces.Worker {
	wr.Lock()
	defer wr.Unlock()

	availableWorkers := make(map[string]interfaces.Worker)
	for id, worker := range wr.Workers {
		if worker.GetStatus().GetAvailable() {
			availableWorkers[id] = worker
		}
	}

	return availableWorkers
}

func (wr *WorkerRegistry) GetWorkersWithBlocksDetected(
	workers map[string]interfaces.Worker,
	blockId string,
) map[string]interfaces.Worker {
	workersWithBlocks := make(map[string]interfaces.Worker)
	for id, worker := range workers {
		workerBlocks := worker.GetStatus().GetBlocks()
		for _, workerBlockId := range workerBlocks {
			if workerBlockId == blockId {
				workersWithBlocks[id] = worker
			}
		}
	}

	return workersWithBlocks
}

func (wr *WorkerRegistry) GetWorkersWithPipeline(
	workers map[string]interfaces.Worker,
	pipelineSlug string,
) map[string]interfaces.Worker {
	workersWithPipeline := make(map[string]interfaces.Worker)
	for id, worker := range workers {
		workerPipelines, err := wr.GetWorkerPipelines(worker)
		if err != nil {
			continue
		}
		for workerPipelineSlug, _ := range workerPipelines {
			if workerPipelineSlug == pipelineSlug {
				workersWithPipeline[id] = worker
			}
		}
	}

	return workersWithPipeline
}

func (wr *WorkerRegistry) GetWorkersWithBlocksAvailable(
	workers map[string]interfaces.Worker,
	blockId string,
) map[string]interfaces.Worker {
	workersWithBlocksAvailable := make(map[string]interfaces.Worker)
	for id, worker := range workers {
		workerBlocks, err := wr.GetWorkerBlocks(worker)
		if err != nil {
			continue
		}
		for workerBlockId, workerBlock := range workerBlocks {
			if workerBlockId != blockId {
				continue
			}
			if workerBlockMap, ok := workerBlock.(map[string]interface{}); ok {
				if _, ok := workerBlockMap["available"]; !ok {
					continue
				}
				if workerBlockMap["available"].(bool) {
					workersWithBlocksAvailable[id] = worker
				}
			}
		}
	}

	return workersWithBlocksAvailable
}

func (wr *WorkerRegistry) GetValidWorkers(
	pipelineSlug string,
	blockId string,
) map[string]interfaces.Worker {
	availableWorkers := wr.GetAvailableWorkers()
	workersWithBlocks := wr.GetWorkersWithBlocksDetected(availableWorkers, blockId)
	workersWithPipeline := wr.GetWorkersWithPipeline(workersWithBlocks, pipelineSlug)
	workersWithBlocksAndPipeline := wr.GetWorkersWithBlocksAvailable(workersWithPipeline, blockId)

	return workersWithBlocksAndPipeline
}

func (wr *WorkerRegistry) QueryWorkerAPI(
	worker interfaces.Worker,
	path string,
	method string,
	result interface{},
) error {
	request, err := http.NewRequest(
		method,
		fmt.Sprintf(
			"%s/%s",
			worker.GetAPIEndpoint(),
			path,
		),
		nil, // Use a different io.Reader if you need to send a body
	)
	if err != nil {
		return err
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"worker API returned status code %d",
			response.StatusCode,
		)
	}

	err = json.NewDecoder(response.Body).Decode(result)
	if err != nil {
		return err
	}

	return nil
}

func (wr *WorkerRegistry) GetWorkerPipelines(worker interfaces.Worker) (map[string]interface{}, error) {
	pipelines := make(map[string]interface{})
	err := wr.QueryWorkerAPI(worker, "pipelines", "GET", &pipelines)
	return pipelines, err
}

func (wr *WorkerRegistry) GetWorkerBlocks(worker interfaces.Worker) (map[string]interface{}, error) {
	blocks := make(map[string]interface{})
	err := wr.QueryWorkerAPI(worker, "blocks", "GET", &blocks)
	return blocks, err
}

func (wr *WorkerRegistry) ResumeProcessingAtWorker(
	worker interfaces.Worker,
	pipelineSlug string,
	processingId uuid.UUID,
	inputData schemas.PipelineStartInputSchema,
) error {
	var pipelineResumeOutput schemas.PipelineResumeOutputSchema
	if err := wr.QueryWorkerAPI(
		worker,
		fmt.Sprintf(
			"pipelines/%s/resume",
			pipelineSlug,
		),
		"POST",
		&pipelineResumeOutput,
	); err != nil {
		return err
	}

	if pipelineResumeOutput.ProcessingID != processingId {
		return fmt.Errorf(
			"worker %s returned different processing ID %s",
			worker.GetId(),
			pipelineResumeOutput.ProcessingID,
		)
	}

	return nil
}
