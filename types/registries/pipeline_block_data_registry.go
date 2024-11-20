package registries

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

const (
	OUTPUT_FILE_TEMPLATE       = "output_%d"
	OUTPUT_FILE_TEMPLATE_REGEX = "output_\\d+"
	LOG_FILE_TEMPLATE          = "log_%d"
	LOG_FILE_TEMPLATE_REGEX    = "log_\\d+"
	STATUS_FILE_TEMPLATE       = "status_%d"
	STATUS_FILE_TEMPLATE_REGEX = "status_\\d+"
)

var (
	OUTPUT_FILE_REGEX = regexp.MustCompile(OUTPUT_FILE_TEMPLATE_REGEX)
	LOG_FILE_REGEX    = regexp.MustCompile(LOG_FILE_TEMPLATE_REGEX)
	STATUS_FILE_REGEX = regexp.MustCompile(STATUS_FILE_TEMPLATE_REGEX)
)

type PipelineBlockDataRegistry struct {
	sync.Mutex

	processingId uuid.UUID
	pipelineSlug string
	storages     []interfaces.Storage

	// TODO: Follow Registry interface and replace []*bytes.Buffer
	//	to interfaces.StorageLocation
	pipelineBlockData map[string][]*bytes.Buffer
}

// Ensure PipelineBlockDataRegistry implements the PipelineBlockDataRegistry
var _ interfaces.PipelineBlockDataRegistry = (*PipelineBlockDataRegistry)(nil)

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

func (r *PipelineBlockDataRegistry) Add(input []*bytes.Buffer) {
}

func (r *PipelineBlockDataRegistry) PrepareBlockData(blockSlug string, size int) {
	r.Lock()
	defer r.Unlock()

	// if array does not exist, create it
	if _, ok := r.pipelineBlockData[blockSlug]; !ok {
		r.pipelineBlockData[blockSlug] = make([]*bytes.Buffer, size)
	}

	// if array is not long enough, copy and extend it
	if size > len(r.pipelineBlockData[blockSlug]) {
		newData := make([]*bytes.Buffer, size)
		copy(newData, r.pipelineBlockData[blockSlug])
		r.pipelineBlockData[blockSlug] = newData
	}
}

func (r *PipelineBlockDataRegistry) UpdateBlockData(blockSlug string, index int, data *bytes.Buffer) {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.pipelineBlockData[blockSlug]; !ok {
		r.pipelineBlockData[blockSlug] = make([]*bytes.Buffer, 0)
	}

	// Ensure that the index is within the bounds of the slice
	if index >= 0 && index < len(r.pipelineBlockData[blockSlug]) {
		r.pipelineBlockData[blockSlug][index] = data
	}
}

func (r *PipelineBlockDataRegistry) AddBlockData(blockSlug string, data *bytes.Buffer) {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.pipelineBlockData[blockSlug]; !ok {
		r.pipelineBlockData[blockSlug] = make([]*bytes.Buffer, 0)
	}

	r.pipelineBlockData[blockSlug] = append(r.pipelineBlockData[blockSlug], data)
}

func (r *PipelineBlockDataRegistry) Get(blockSlug string) []*bytes.Buffer {
	r.Lock()
	defer r.Unlock()

	return r.pipelineBlockData[blockSlug]
}

func (r *PipelineBlockDataRegistry) GetAll() map[string][]*bytes.Buffer {
	r.Lock()
	defer r.Unlock()

	return r.pipelineBlockData
}

func (r *PipelineBlockDataRegistry) Delete(blockSlug string) {
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

func (r *PipelineBlockDataRegistry) Shutdown(context context.Context) error {
	r.Lock()
	defer r.Unlock()

	return nil
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
			// TODO: Add respect to file suffix ( output_{i}.<mimetype> )
			data, err := storage.GetObjectBytes(object)
			if err != nil {
				continue
			}

			// If there is only one storage, we can assume that the data is the same
			// otherwise, we only add the data if the storage is minio ( Remote )
			if storage.GetStorageName() == "minio" || len(storages) == 1 {
				r.AddBlockData(blockSlug, data)
			}
		}
	}

	return r.Get(blockSlug)
}

// SavePipelineLog saves the Pipeline Execution Log & Status
func (r *PipelineBlockDataRegistry) SavePipelineLog(
	logBuffer *config.SafeBuffer,
	logFileConstructor func(id uuid.UUID, slug string, logId uuid.UUID, logBufferFS *bytes.Buffer, storage interfaces.Storage) interfaces.PipelineProcessingDetails,
	statusClassConstructor func(id uuid.UUID, slug string, logId uuid.UUID, logBufferCS *bytes.Buffer, storage interfaces.Storage) interfaces.PipelineProcessingStatus,
) {
	r.Lock()
	defer r.Unlock()

	logger := config.GetLogger()
	filePath := fmt.Sprintf(
		"%s/%s",
		r.pipelineSlug,
		r.processingId,
	)
	logBytes := logBuffer.Bytes()
	logBuffer.Reset()

	// convert current date to timestamp
	logIndex := time.Now().Unix()
	logId := uuid.New()

	for _, storage := range r.storages {
		logFileContent := bytes.NewBuffer(logBytes)

		// LOG file
		logStorageLocation := storage.NewStorageLocation(
			path.Join(
				filePath,
				fmt.Sprintf(LOG_FILE_TEMPLATE, logIndex),
			),
		)
		if logContent, err := json.Marshal(
			logFileConstructor(
				r.processingId,
				r.pipelineSlug,
				logId,
				bytes.NewBuffer(logBytes),
				storage,
			),
		); err == nil {
			logFileContent = bytes.NewBuffer(logContent)
		}

		if _, err := storage.PutObjectBytes(logStorageLocation, logFileContent); err != nil {
			logger.Error(err)
		}

		// STATUS file
		statusStorageLocation := storage.NewStorageLocation(
			path.Join(
				filePath,
				fmt.Sprintf(STATUS_FILE_TEMPLATE, logIndex),
			),
		)
		statusContent, err := json.Marshal(statusClassConstructor(r.processingId, r.pipelineSlug, logId, bytes.NewBuffer(logBytes), storage))
		if err != nil {
			logger.Error(err)
			continue
		}
		if _, err := storage.PutObjectBytes(
			statusStorageLocation,
			bytes.NewBuffer(statusContent),
		); err != nil {
			logger.Error(err)
		}
	}
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
		if dataCopy.Len() == 0 {
			dataCopy = bytes.NewBuffer([]byte("null"))
		}

		destinationStorageLocation, err := storage.PutObjectBytes(
			storage.NewStorageLocation(
				path.Join(
					filePath,
					fmt.Sprintf(OUTPUT_FILE_TEMPLATE, outputIndex),
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
