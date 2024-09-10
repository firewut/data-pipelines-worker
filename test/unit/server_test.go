package unit_test

import (
	"data-pipelines-worker/api"
)

func (suite *UnitTestSuite) TestNewServer() {
	server := api.NewServer()

	suite.Equal(suite._config, server.GetConfig())
}
