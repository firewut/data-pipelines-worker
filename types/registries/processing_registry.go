package registries

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"data-pipelines-worker/types/interfaces"
)

const (
	numReaders                 int = 10
	processingCompletedBufSize int = 100
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
	notificationChannel        chan interfaces.Processing // Channel for external notifications
}

// Ensure ProcessingRegistry implements the ProcessingRegistry
var _ interfaces.ProcessingRegistry = (*ProcessingRegistry)(nil)

func NewProcessingRegistry() *ProcessingRegistry {
	registry := &ProcessingRegistry{
		Processing:                 make(map[uuid.UUID]interfaces.Processing),
		processingCompletedChannel: make(chan interfaces.Processing, processingCompletedBufSize),
		notificationChannel:        nil,
	}

	for i := 0; i < numReaders; i++ {
		go registry.processCompleted()
	}

	return registry
}

func (pr *ProcessingRegistry) processCompleted() {

	for processing := range pr.processingCompletedChannel {
		// Copy the reference outside the lock
		var notificationChannel chan interfaces.Processing
		pr.Lock()
		notificationChannel = pr.notificationChannel
		pr.Unlock()

		// Send to the notification channel if it's set
		if notificationChannel != nil {
			notificationChannel <- processing
		}
	}
}

func (pr *ProcessingRegistry) SetNotificationChannel(channel chan interfaces.Processing) {
	pr.Lock()
	defer pr.Unlock()
	pr.notificationChannel = channel
}

func (pr *ProcessingRegistry) Add(p interfaces.Processing) {
	pr.Lock()
	defer pr.Unlock()

	pr.Processing[p.GetId()] = p
	p.SetRegistryNotificationChannel(pr.processingCompletedChannel)
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
		shutdownWg.Wait()
		close(pr.processingCompletedChannel)
		close(done)
		if pr.notificationChannel != nil {
			close(pr.notificationChannel)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (pr *ProcessingRegistry) StartProcessing(processing interfaces.Processing) (
	interfaces.ProcessingOutput,
	bool,
	error,
) {
	pr.Add(processing)
	return processing.Start()
}

func (pr *ProcessingRegistry) GetProcessingCompletedChannel() chan interfaces.Processing {
	return pr.processingCompletedChannel
}
