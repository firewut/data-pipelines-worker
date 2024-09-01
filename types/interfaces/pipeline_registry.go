package interfaces

type PipelineCatalogueLoader interface {
	Load(Storage, string) (map[string]Pipeline, error)
}
