package dataclasses

import (
	"os"
	"path/filepath"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

type PipelineCatalogueLoader struct {
}

func (pcl *PipelineCatalogueLoader) Load(
	storage interfaces.Storage,
	cataloguePath string,
) (
	map[string]interfaces.Pipeline,
	error,
) {
	// TODO: Respect storage
	_config := config.GetConfig()
	pipelines := make(map[string]interfaces.Pipeline)

	// List all files in the directory
	files, err := os.ReadDir(_config.Pipeline.Catalogue)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(_config.Pipeline.Catalogue, file.Name())

		if fileContent, err := os.ReadFile(filePath); err == nil {
			if pipeline, err := NewPipelineFromCatalogue(fileContent); err == nil {
				pipelines[pipeline.GetSlug()] = pipeline
			} else {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	return pipelines, nil
}
