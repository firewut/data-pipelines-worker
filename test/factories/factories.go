package factories

import (
	"data-pipelines-worker/api"
)

func ServerFactory() *api.Server {
	return api.NewServer()
}
