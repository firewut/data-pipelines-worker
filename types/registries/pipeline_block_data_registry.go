package registries

import (
	"bytes"
	"fmt"
	"path"
	"sync"

	"github.com/google/uuid"

	"data-pipelines-worker/types/interfaces"
)

type PipelineBlockDataRegistry struct {
	sync.Mutex

	processingId      uuid.UUID
	pipelineSlug      string
	storages          []interfaces.Storage
	pipelineBlockData map[string][]*bytes.Buffer
}

type PipelineBlockDataRegistrySavedOutput struct {
	StorageLocation interfaces.StorageLocation
	Error           error
}

func NewPipelineBlockDataRegistry(
	processingId uuid.UUID,
	pipelineSlug string,
	storages []interfaces.Storage,
) *PipelineBlockDataRegistry {
	registry := &PipelineBlockDataRegistry{
		pipelineBlockData: make(map[string][]*bytes.Buffer),
		processingId:      processingId,
		pipelineSlug:      pipelineSlug,
		storages:          storages,
	}

	return registry
}

func (r *PipelineBlockDataRegistry) Add(
	blockSlug string,
	data *bytes.Buffer,
) {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.pipelineBlockData[blockSlug]; !ok {
		r.pipelineBlockData[blockSlug] = make([]*bytes.Buffer, 0)
	}

	r.pipelineBlockData[blockSlug] = append(r.pipelineBlockData[blockSlug], data)
}

func (r *PipelineBlockDataRegistry) Get(
	blockSlug string,
) []*bytes.Buffer {
	r.Lock()
	defer r.Unlock()

	return r.pipelineBlockData[blockSlug]
}

func (r *PipelineBlockDataRegistry) GetAll() map[string][]*bytes.Buffer {
	r.Lock()
	defer r.Unlock()

	return r.pipelineBlockData
}

func (r *PipelineBlockDataRegistry) Delete(
	blockSlug string,
) {
	r.Lock()
	defer r.Unlock()

	delete(r.pipelineBlockData, blockSlug)
}

func (r *PipelineBlockDataRegistry) DeleteAll() {
	r.Lock()
	defer r.Unlock()

	for blockSlug := range r.pipelineBlockData {
		delete(r.pipelineBlockData, blockSlug)
	}
}

func (r *PipelineBlockDataRegistry) Shutdown() {
}

func (r *PipelineBlockDataRegistry) GetProcessingId() uuid.UUID {
	r.Lock()
	defer r.Unlock()

	return r.processingId
}

func (r *PipelineBlockDataRegistry) GetPipelineSlug() string {
	r.Lock()
	defer r.Unlock()

	return r.pipelineSlug
}

func (r *PipelineBlockDataRegistry) GetStorages() []interfaces.Storage {
	r.Lock()
	defer r.Unlock()

	return r.storages
}

func (r *PipelineBlockDataRegistry) AddStorage(storage interfaces.Storage) {
	r.Lock()
	defer r.Unlock()

	r.storages = append(r.storages, storage)
}

func (r *PipelineBlockDataRegistry) SetStorages(storages []interfaces.Storage) {
	r.Lock()
	defer r.Unlock()

	r.storages = storages
}

func (r *PipelineBlockDataRegistry) LoadOutput(blockSlug string) []*bytes.Buffer {
	// Warning: This is relative to the storage Bucket and LocalDirectory
	blockSlugOutputCatalogue := path.Join(
		r.pipelineSlug,
		r.processingId.String(),
		blockSlug,
	)

	storages := r.GetStorages()
	for _, storage := range storages {
		blockDataLocation := storage.NewStorageLocation(blockSlugOutputCatalogue)
		objects, err := storage.ListObjects(blockDataLocation)

		if err != nil || len(objects) == 0 {
			continue
		}

		for _, object := range objects {
			data, err := storage.GetObjectBytes(object)
			if err != nil {
				fmt.Println(">>", object, " | ", err)
				continue
			}

			r.Add(blockSlug, data)
		}
	}

	return r.Get(blockSlug)
}

// SaveOutput saves the output to all storages
func (r *PipelineBlockDataRegistry) SaveOutput(
	blockSlug string,
	outputIndex int,
	output *bytes.Buffer,
) []PipelineBlockDataRegistrySavedOutput {
	// Generates is a file named:
	// <pipeline-slug>/<processing-id>/<block-slug>/output_{i}.<mimetype>

	result := make([]PipelineBlockDataRegistrySavedOutput, 0)

	filePath := fmt.Sprintf(
		"%s/%s/%s",
		r.pipelineSlug,
		r.processingId,
		blockSlug,
	)

	for _, storage := range r.storages {
		dataCopy := bytes.NewBuffer(output.Bytes())

		destinationStorageLocation, err := storage.PutObjectBytes(
			storage.NewStorageLocation(
				path.Join(
					filePath,
					fmt.Sprintf("output_%d", outputIndex),
				),
			),
			dataCopy,
		)

		result = append(
			result,
			PipelineBlockDataRegistrySavedOutput{
				StorageLocation: destinationStorageLocation,
				Error:           err,
			},
		)
	}

	return result
}
