package interfaces

import (
	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"
)

type PipelineProcessor interface {
	Process() int
}

type Pipeline interface {
	GetID() uuid.UUID
	GetSlug() string
	GetTitle() string
	GetDescription() string
	GetBlocks() []ProcessableBlockData
	GetSchemaString() string
	GetSchemaPtr() *gojsonschema.Schema
}
