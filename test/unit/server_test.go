package unit_test

import (
	"data-pipelines-worker/api"
	"data-pipelines-worker/types/config"
)

func (suite *UnitTestSuite) TestNewServer() {
	server := api.NewServer(config.GetConfig())

	suite.Equal(suite._config, server.GetConfig())
}
