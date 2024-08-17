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
	suite.EqualValues(0.0, config.DNSSD.Load)
	suite.Equal(false, config.DNSSD.Available)

	suite.NotEmpty(config.Storage.CredentialsPath)
	suite.NotEmpty(config.Storage.AccessKey)
	suite.NotEmpty(config.Storage.Api)
	suite.NotEmpty(config.Storage.Path)
	suite.NotEmpty(config.Storage.SecretKey)
	suite.NotEmpty(config.Storage.Url)
}
