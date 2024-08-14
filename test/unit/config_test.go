package unit_test

import (
	"data-pipelines-worker/types"
)

func (suite *UnitTestSuite) TestNewConfig() {
	config := types.NewConfig()

	suite.Equal("debug", config.Log.Level)
	suite.Equal("0.0.0.0", config.HTTPAPIServer.Host)
	suite.Equal(8080, config.HTTPAPIServer.Port)

	suite.Equal("data-pipelines-worker", config.DNSSD.ServiceName)
	suite.Equal("_http._tcp.", config.DNSSD.ServiceType)
	suite.Equal("local.", config.DNSSD.ServiceDomain)
	suite.Equal(8080, config.DNSSD.ServicePort)
	suite.Equal("", config.DNSSD.Load)
	suite.Equal(false, config.DNSSD.Available)
}
