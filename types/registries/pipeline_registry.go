package registries

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

type PipelineRegistry struct {
	sync.Mutex

	Pipelines               map[string]interfaces.Pipeline
	pipelineCatalogueLoader interfaces.PipelineCatalogueLoader
	storage                 interfaces.Storage
}

func NewPipelineRegistry(pipelineCatalogueLoader interfaces.PipelineCatalogueLoader) (*PipelineRegistry, error) {
	_config := config.GetConfig()

	registry := &PipelineRegistry{
		Pipelines:               make(map[string]interfaces.Pipeline),
		pipelineCatalogueLoader: pipelineCatalogueLoader,
		storage:                 pipelineCatalogueLoader.GetStorage(),
	}

	pipelines, err := pipelineCatalogueLoader.LoadCatalogue(
		_config.Pipeline.Catalogue,
	)
	if err != nil {
		return nil, err
	}

	for _, pipeline := range pipelines {
		registry.Register(pipeline)
	}

	return registry, nil
}

func (pr *PipelineRegistry) Register(p interfaces.Pipeline) {
	_config := config.GetConfig()
	registrySchema := _config.Pipeline.SchemaPtr
	pipelineSchemaLoader := gojsonschema.NewStringLoader(p.GetSchemaString())
	validationResult, err := registrySchema.Validate(pipelineSchemaLoader)

	if err != nil {
		panic(err)
	}
	if !validationResult.Valid() {
		errStr := fmt.Sprintf("Pipeline schema is invalid for pipeline: %s", p.GetSlug())
		for _, err := range validationResult.Errors() {
			errStr += fmt.Sprintf("\n- %s", err)
		}
		panic(errStr)
	}

	pr.Lock()
	defer pr.Unlock()

	pr.Pipelines[p.GetSlug()] = p
}

func (pr *PipelineRegistry) SetStorage(storage interfaces.Storage) {
	pr.Lock()
	defer pr.Unlock()

	pr.storage = storage
}

func (pr *PipelineRegistry) GetStorage() interfaces.Storage {
	pr.Lock()
	defer pr.Unlock()

	return pr.storage
}

func (pr *PipelineRegistry) Get(slug string) interfaces.Pipeline {
	pr.Lock()
	defer pr.Unlock()

	return pr.Pipelines[slug]
}

func (pr *PipelineRegistry) GetAll() map[string]interfaces.Pipeline {
	pr.Lock()
	defer pr.Unlock()

	return pr.Pipelines
}

func (pr *PipelineRegistry) Delete(slug string) {
	pr.Lock()
	defer pr.Unlock()

	delete(pr.Pipelines, slug)
}

func (pr *PipelineRegistry) Shutdown() {
}

func (pr *PipelineRegistry) StartPipeline(
	data schemas.PipelineStartInputSchema,
) (uuid.UUID, error) {
	pipeline := pr.Get(data.Pipeline.Slug)
	if pipeline == nil {
		return uuid.UUID{}, fmt.Errorf("pipeline with slug %s not found", data.Pipeline.Slug)
	}

	return pipeline.StartProcessing(data, pr.GetStorage())
}
