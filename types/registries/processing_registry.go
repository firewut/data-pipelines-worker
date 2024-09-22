package registries

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"data-pipelines-worker/types/interfaces"
)

var (
	onceProcessingRegistry     sync.Once
	processingRegistryInstance *ProcessingRegistry
)

func GetProcessingRegistry(forceNewInstance ...bool) *ProcessingRegistry {
	if len(forceNewInstance) > 0 && forceNewInstance[0] {
		newInstance := NewProcessingRegistry()
		processingRegistryInstance = newInstance
		onceProcessingRegistry = sync.Once{}
		return newInstance
	}

	onceProcessingRegistry.Do(func() {
		processingRegistryInstance = NewProcessingRegistry()
	})

	return processingRegistryInstance
}

// ProcessingRegistry is a registry for Processing entries
type ProcessingRegistry struct {
	sync.Mutex

	Processing                 map[uuid.UUID]interfaces.Processing
	processingCompletedChannel chan interfaces.Processing
}

func NewProcessingRegistry() *ProcessingRegistry {
	registry := &ProcessingRegistry{
		Processing:                 make(map[uuid.UUID]interfaces.Processing),
		processingCompletedChannel: make(chan interfaces.Processing),
	}

	return registry
}

func (pr *ProcessingRegistry) Add(p interfaces.Processing) {
	pr.Lock()

	p.SetRegistryNotificationChannel(pr.GetProcessingCompletedChannel())
	pr.Processing[p.GetId()] = p

	pr.Unlock()
}

func (pr *ProcessingRegistry) Get(id string) interfaces.Processing {
	pr.Lock()
	defer pr.Unlock()

	return pr.Processing[uuid.MustParse(id)]
}

func (pr *ProcessingRegistry) GetAll() map[string]interfaces.Processing {
	pr.Lock()
	defer pr.Unlock()

	result := make(map[string]interfaces.Processing)
	for id, processing := range pr.Processing {
		result[id.String()] = processing
	}

	return result
}

func (pr *ProcessingRegistry) Delete(id string) {
	pr.Lock()
	defer pr.Unlock()

	delete(pr.Processing, uuid.MustParse(id))
}

func (pr *ProcessingRegistry) DeleteAll() {
	pr.Lock()
	defer pr.Unlock()

	for id := range pr.Processing {
		delete(pr.Processing, id)
	}
}

func (pr *ProcessingRegistry) Shutdown(ctx context.Context) error {
	pr.Lock()
	defer close(pr.processingCompletedChannel)
	defer pr.Unlock()

	shutdownWg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, processing := range pr.Processing {
		shutdownWg.Add(1)
		go func(_processing interfaces.Processing) {
			defer shutdownWg.Done()
			_processing.Shutdown(ctx)
		}(processing)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		shutdownWg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (pr *ProcessingRegistry) StartProcessingById(processingId uuid.UUID) (
	interfaces.ProcessingOutput,
	error,
) {
	pr.Lock()
	processing := pr.Processing[processingId]
	pr.Unlock()

	return processing.Start()
}

func (pr *ProcessingRegistry) StartProcessing(processing interfaces.Processing) (
	interfaces.ProcessingOutput,
	error,
) {
	pr.Add(processing)
	return pr.StartProcessingById(processing.GetId())
}

func (pr *ProcessingRegistry) GetProcessingCompletedChannel() chan interfaces.Processing {
	return pr.processingCompletedChannel
}
