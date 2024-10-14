package unit_test

import (
	"time"

	"data-pipelines-worker/types/config"
)

func (suite *UnitTestSuite) TestGetConfig() {
	_config := config.GetConfig()

	suite.Equal("info", _config.Log.Level)
	suite.Equal("0.0.0.0", _config.HTTPAPIServer.Host)
	suite.Equal(8080, _config.HTTPAPIServer.Port)

	suite.Equal("data-pipelines-worker", _config.DNSSD.ServiceName)
	suite.Equal("_http._tcp.", _config.DNSSD.ServiceType)
	suite.Equal("local.", _config.DNSSD.ServiceDomain)
	suite.Equal(8080, _config.DNSSD.ServicePort)
	suite.EqualValues(0.0, _config.DNSSD.Load)
	suite.Equal(false, _config.DNSSD.Available)

	suite.NotEmpty(_config.Storage.Local.RootPath)
	suite.Equal("/tmp", _config.Storage.Local.RootPath)

	suite.NotEmpty(_config.Storage.Minio.CredentialsPath)
	suite.NotEmpty(_config.Storage.Minio.AccessKey)
	suite.NotEmpty(_config.Storage.Minio.Api)
	suite.NotEmpty(_config.Storage.Minio.Path)
	suite.NotEmpty(_config.Storage.Minio.SecretKey)
	suite.NotEmpty(_config.Storage.Minio.Url)

	suite.NotEmpty(_config.Pipeline.StoragePath)
	suite.NotNil(_config.Pipeline.SchemaPtr)

	suite.NotEmpty(_config.Telegram.CredentialsPath)
	suite.NotEmpty(_config.Telegram.Token)

	suite.NotEmpty(_config.OpenAI.CredentialsPath)
	suite.NotEmpty(_config.OpenAI.Token)

	suite.NotEmpty(_config.Blocks)
	suite.NotEmpty(_config.Blocks["http_request"])
	suite.NotEmpty(_config.Blocks["http_request"].Detector)
	suite.Equal(
		_config.Blocks["http_request"].Detector.CheckInterval,
		time.Duration(1*time.Minute),
	)

	suite.Contains(
		_config.Blocks["http_request"].Detector.Conditions["url"],
		"http://",
	)
	suite.NotEmpty(
		_config.Blocks["http_request"].Reliability,
	)
	suite.Equal(
		"exponential_backoff",
		_config.Blocks["http_request"].Reliability.Policy,
	)
	suite.Equal(
		config.BlockConfigReliabilityExponentialBackoff{
			MaxRetries: 5,
			RetryDelay: 1,
			RetryCodes: []int{500, 502, 503, 504},
		},
		_config.Blocks["http_request"].Reliability.PolicyConfig,
	)
}
