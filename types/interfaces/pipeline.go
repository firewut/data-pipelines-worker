package interfaces

import (
	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"

	"data-pipelines-worker/api/schemas"
)

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
}

type PipelineCatalogueLoader interface {
	SetStorage(Storage)
	GetStorage() Storage

	LoadCatalogue(string) (map[string]Pipeline, error)
}
