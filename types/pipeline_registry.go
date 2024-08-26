package types

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

type PipelineRegistry struct {
	sync.Mutex

	Pipelines map[string]*Pipeline
}

func NewPipelineRegistry() *PipelineRegistry {
	return &PipelineRegistry{
		Pipelines: make(map[string]*Pipeline),
	}
}

func (pr *PipelineRegistry) Register(p *Pipeline) {
	config := GetConfig()
	registrySchema := config.Pipeline.SchemaPtr
	pipelineSchemaLoader := gojsonschema.NewStringLoader(p.schemaString)
	validationResult, err := registrySchema.Validate(pipelineSchemaLoader)

	pr.Lock()
	defer pr.Unlock()

	if err != nil {
		panic(err)
	}
	if !validationResult.Valid() {
		errStr := fmt.Sprintf("Pipeline schema is invalid for pipeline: %s", p.Slug)
		for _, err := range validationResult.Errors() {
			errStr += fmt.Sprintf("\n- %s", err)
		}
		panic(errStr)
	}

	pr.Pipelines[p.Slug] = p
}

func (pr *PipelineRegistry) Get(slug string) *Pipeline {
	pr.Lock()
	defer pr.Unlock()

	return pr.Pipelines[slug]
}

func (pr *PipelineRegistry) GetAll() map[string]*Pipeline {
	pr.Lock()
	defer pr.Unlock()

	return pr.Pipelines
}

func (pr *PipelineRegistry) Delete(slug string) {
	pr.Lock()
	defer pr.Unlock()

	delete(pr.Pipelines, slug)
}

func (pr *PipelineRegistry) LoadFromCatalogue() {
	config := GetConfig()

	// List all files in the directory
	files, err := os.ReadDir(config.Pipeline.Catalogue)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(config.Pipeline.Catalogue, file.Name())

		if fileContent, err := os.ReadFile(filePath); err == nil {
			pipeline := NewPipeline(fileContent)
			pr.Register(pipeline)
		} else {
			panic(err)
		}
	}

}
