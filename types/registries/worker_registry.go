package registries

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/config"
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

// Ensure WorkerRegistry implements the WorkerRegistry
var _ interfaces.WorkerRegistry = (*WorkerRegistry)(nil)

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
	body interface{},
	result interface{},
) (string, error) {
	var requestBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonBody)
	} else {
		requestBody = nil
	}

	request, err := http.NewRequest(
		method,
		fmt.Sprintf(
			"%s/%s",
			worker.GetAPIEndpoint(),
			path,
		),
		requestBody,
	)
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	responseBodyBytes, _ := io.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		responseString := string(responseBodyBytes)
		config.GetLogger().Errorf(
			"worker API returned status code %d. Response was %s",
			response.StatusCode,
			responseString,
		)

		return responseString, fmt.Errorf(
			"worker API returned status code %d",
			response.StatusCode,
		)
	}

	if err := json.Unmarshal(responseBodyBytes, &result); err != nil {
		responseString := string(responseBodyBytes)

		config.GetLogger().Errorf(
			"failed to unmarshal worker API response: %s. Response was %s",
			err.Error(),
			responseString,
		)
		return responseString, err
	}

	return "", nil
}

func (wr *WorkerRegistry) GetWorkerPipelines(worker interfaces.Worker) (map[string]interface{}, error) {
	pipelines := make(map[string]interface{})
	_, err := wr.QueryWorkerAPI(worker, "pipelines", "GET", nil, &pipelines)
	return pipelines, err
}

func (wr *WorkerRegistry) GetWorkerBlocks(worker interfaces.Worker) (map[string]interface{}, error) {
	blocks := make(map[string]interface{})
	_, err := wr.QueryWorkerAPI(worker, "blocks", "GET", nil, &blocks)
	return blocks, err
}

func (wr *WorkerRegistry) ResumeProcessingAtWorker(
	worker interfaces.Worker,
	pipelineSlug string,
	processingId uuid.UUID,
	inputData schemas.PipelineStartInputSchema,
) error {
	var pipelineResumeOutput schemas.PipelineResumeOutputSchema
	if _, err := wr.QueryWorkerAPI(
		worker,
		fmt.Sprintf(
			"pipelines/%s/resume",
			pipelineSlug,
		),
		"POST",
		inputData,
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
