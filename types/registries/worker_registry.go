package registries

import (
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
	blockSlug string,
) error {
	wr.Lock()
	defer wr.Unlock()

	return nil
}
