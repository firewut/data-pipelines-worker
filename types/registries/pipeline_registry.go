package registries

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"

	"data-pipelines-worker/api/schemas"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

// PipelineRegistry is a registry for Pipelines from Catalogue
type PipelineRegistry struct {
	sync.Mutex

	Pipelines               map[string]interfaces.Pipeline
	pipelineResultStorages  []interfaces.Storage
	pipelineCatalogueLoader interfaces.PipelineCatalogueLoader

	workerRegistry     interfaces.WorkerRegistry
	blockRegistry      interfaces.BlockRegistry
	processingRegistry interfaces.ProcessingRegistry
}

// Ensure PipelineRegistry implements the PipelineRegistry
var _ interfaces.PipelineRegistry = (*PipelineRegistry)(nil)

func NewPipelineRegistry(
	workerRegistry interfaces.WorkerRegistry,
	blockRegistry interfaces.BlockRegistry,
	processingRegistry interfaces.ProcessingRegistry,
	pipelineCatalogueLoader interfaces.PipelineCatalogueLoader,
) (*PipelineRegistry, error) {
	_config := config.GetConfig()

	registry := &PipelineRegistry{
		workerRegistry:          workerRegistry,
		blockRegistry:           blockRegistry,
		processingRegistry:      processingRegistry,
		Pipelines:               make(map[string]interfaces.Pipeline),
		pipelineResultStorages:  make([]interfaces.Storage, 0),
		pipelineCatalogueLoader: pipelineCatalogueLoader,
	}

	pipelines, err := pipelineCatalogueLoader.LoadCatalogue(
		_config.Pipeline.Catalogue,
	)
	if err != nil {
		return nil, err
	}

	for _, pipeline := range pipelines {
		registry.Add(pipeline)
	}

	return registry, nil
}

func (pr *PipelineRegistry) Add(p interfaces.Pipeline) {
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

func (pr *PipelineRegistry) SetPipelineResultStorages(storages []interfaces.Storage) {
	pr.Lock()
	defer pr.Unlock()

	pr.pipelineResultStorages = storages
}

func (pr *PipelineRegistry) GetPipelineResultStorages() []interfaces.Storage {
	pr.Lock()
	defer pr.Unlock()

	return pr.pipelineResultStorages
}

func (pr *PipelineRegistry) AddPipelineResultStorage(storage interfaces.Storage) {
	pr.Lock()
	defer pr.Unlock()

	pr.pipelineResultStorages = append(pr.pipelineResultStorages, storage)
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

func (pr *PipelineRegistry) DeleteAll() {
	pr.Lock()
	defer pr.Unlock()

	for slug := range pr.Pipelines {
		delete(pr.Pipelines, slug)
	}
}

func (pr *PipelineRegistry) Shutdown(ctx context.Context) error {
	return nil
}

func (pr *PipelineRegistry) GetProcessingsStatus(p interfaces.Pipeline) map[uuid.UUID][]interfaces.PipelineProcessingStatus {
	return p.GetProcessingsStatus(
		pr.GetPipelineResultStorages(),
	)
}

func (pr *PipelineRegistry) GetProcessingDetails(p interfaces.Pipeline, pipelineId uuid.UUID) []interfaces.PipelineProcessingDetails {
	return p.GetProcessingDetails(pipelineId, pr.GetPipelineResultStorages())
}

func (pr *PipelineRegistry) StartPipeline(
	data schemas.PipelineStartInputSchema,
) (uuid.UUID, error) {
	pipeline := pr.Get(data.Pipeline.Slug)
	if pipeline == nil {
		return uuid.UUID{}, fmt.Errorf("pipeline with slug %s not found", data.Pipeline.Slug)
	}

	return pipeline.Process(
		pr.GetWorkerRegistry(),
		pr.GetBlockRegistry(),
		pr.GetProcessingRegistry(),
		data,
		pr.GetPipelineResultStorages(),
	)
}

func (pr *PipelineRegistry) ResumePipeline(
	data schemas.PipelineStartInputSchema,
) (uuid.UUID, error) {
	pipeline := pr.Get(data.Pipeline.Slug)
	if pipeline == nil {
		return uuid.UUID{}, fmt.Errorf("pipeline with slug %s not found", data.Pipeline.Slug)
	}

	return pipeline.Process(
		pr.GetWorkerRegistry(),
		pr.GetBlockRegistry(),
		pr.GetProcessingRegistry(),
		data,
		pr.GetPipelineResultStorages(),
	)
}

func (pr *PipelineRegistry) GetWorkerRegistry() interfaces.WorkerRegistry {
	return pr.workerRegistry
}

func (pr *PipelineRegistry) GetBlockRegistry() interfaces.BlockRegistry {
	return pr.blockRegistry
}

func (pr *PipelineRegistry) GetProcessingRegistry() interfaces.ProcessingRegistry {
	return pr.processingRegistry
}
