package generics

import (
	"data-pipelines-worker/types/config"
)

type ConfigurableBlock[T any] interface {
	GetBlockConfig(config.Config) *T
}
