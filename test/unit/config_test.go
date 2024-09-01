package unit_test

import "data-pipelines-worker/types/config"

func (suite *UnitTestSuite) TestGetConfig() {
	_config := config.GetConfig()

	suite.Equal("debug", _config.Log.Level)
	suite.Equal("0.0.0.0", _config.HTTPAPIServer.Host)
	suite.Equal(8080, _config.HTTPAPIServer.Port)

	suite.Equal("data-pipelines-worker", _config.DNSSD.ServiceName)
	suite.Equal("_http._tcp.", _config.DNSSD.ServiceType)
	suite.Equal("local.", _config.DNSSD.ServiceDomain)
	suite.Equal(8080, _config.DNSSD.ServicePort)
	suite.EqualValues(0.0, _config.DNSSD.Load)
	suite.Equal(false, _config.DNSSD.Available)

	suite.NotEmpty(_config.Storage.CredentialsPath)
	suite.NotEmpty(_config.Storage.AccessKey)
	suite.NotEmpty(_config.Storage.Api)
	suite.NotEmpty(_config.Storage.Path)
	suite.NotEmpty(_config.Storage.SecretKey)
	suite.NotEmpty(_config.Storage.Url)

	suite.NotEmpty(_config.Pipeline.StoragePath)
	suite.NotNil(_config.Pipeline.SchemaPtr)

	suite.NotEmpty(_config.OpenAI.CredentialsPath)
	suite.NotEmpty(_config.OpenAI.Token)
}
