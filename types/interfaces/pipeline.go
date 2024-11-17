package interfaces

import (
	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"

	"data-pipelines-worker/api/schemas"
)

type PipelineProcessingStatus interface {
	GetId() uuid.UUID

	MarshalJSON() ([]byte, error)
}

type PipelineProcessingDetails interface {
	GetId() uuid.UUID

	MarshalJSON() ([]byte, error)
}

type Pipeline interface {
	GetId() string
	GetSlug() string
	GetTitle() string
	GetDescription() string
	GetBlocks() []ProcessableBlockData
	GetSchemaString() string
	GetSchemaPtr() *gojsonschema.Schema

	Process(
		WorkerRegistry,
		BlockRegistry,
		ProcessingRegistry,
		schemas.PipelineStartInputSchema,
		[]Storage,
	) (uuid.UUID, error)

	GetProcessingsStatus([]Storage) map[uuid.UUID][]PipelineProcessingStatus
	GetProcessingDetails(uuid.UUID, []Storage) []PipelineProcessingDetails
}

type PipelineCatalogueLoader interface {
	SetStorage(Storage)
	GetStorage() Storage

	LoadCatalogue(string) (map[string]Pipeline, error)
}
