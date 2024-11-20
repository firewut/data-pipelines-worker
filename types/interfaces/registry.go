package interfaces

import (
	"bytes"

	"github.com/google/uuid"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/generics"
)

type RegistryEntry interface {
	GetId() string
	GetValue() interface{}
}

type WorkerRegistry interface {
	generics.Registry[Worker]

	GetAvailableWorkers() map[string]Worker
	GetWorkersWithPipeline(map[string]Worker, string) map[string]Worker
	GetWorkersWithBlocksAvailable(map[string]Worker, string) map[string]Worker
	GetValidWorkers(string, string) map[string]Worker

	GetWorkerPipelines(Worker) (map[string]interface{}, error)
	GetWorkerBlocks(Worker) (map[string]interface{}, error)
	QueryWorkerAPI(Worker, string, string, interface{}, interface{}) (string, error)

	ResumeProcessing(string, uuid.UUID, string, schemas.PipelineStartInputSchema) error
	ResumeProcessingAtWorker(Worker, string, uuid.UUID, schemas.PipelineStartInputSchema) error
}

type PipelineRegistry interface {
	generics.Registry[Pipeline]

	AddPipelineResultStorage(Storage)
	SetPipelineResultStorages([]Storage)
	GetPipelineResultStorages() []Storage

	StartPipeline(schemas.PipelineStartInputSchema) (uuid.UUID, error)
	ResumePipeline(schemas.PipelineStartInputSchema) (uuid.UUID, error)

	GetWorkerRegistry() WorkerRegistry
	GetBlockRegistry() BlockRegistry
	GetProcessingRegistry() ProcessingRegistry

	GetProcessingsStatus(Pipeline) map[uuid.UUID][]PipelineProcessingStatus
	GetProcessingDetails(Pipeline, uuid.UUID) []PipelineProcessingDetails
	GetProcessingDetailsByLogId(Pipeline, uuid.UUID, uuid.UUID) PipelineProcessingDetails
}

type BlockRegistry interface {
	generics.Registry[Block]

	DetectBlocks()
	GetAvailableBlocks() map[string]Block
	IsAvailable(Block) bool
}

type ProcessingRegistry interface {
	generics.Registry[Processing]

	StartProcessing(Processing) ProcessingOutput
	GetProcessingCompletedChannel() chan Processing

	SetNotificationChannel(chan Processing)
}

type PipelineBlockDataRegistry interface {
	generics.Registry[[]*bytes.Buffer]

	AddBlockData(string, *bytes.Buffer)

	PrepareBlockData(string, int)
	UpdateBlockData(string, int, *bytes.Buffer)
}
