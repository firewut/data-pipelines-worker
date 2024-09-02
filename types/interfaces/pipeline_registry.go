package interfaces

type PipelineCatalogueLoader interface {
	SetStorage(Storage)
	GetStorage() Storage

	LoadCatalogue(string) (map[string]Pipeline, error)
}
