package functional_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"

	"data-pipelines-worker/api"
	"data-pipelines-worker/test/factories"
	"data-pipelines-worker/types/config"
)

type FunctionalTestSuite struct {
	sync.RWMutex

	suite.Suite

	server          *api.Server
	_config         config.Config
	httpTestServers []*httptest.Server // to mock http requests
}

func TestFunctionalTestSuite(t *testing.T) {
	// Set Logger level to debug
	config.GetLogger().SetLevel(log.INFO)

	suite.Run(t, new(FunctionalTestSuite))
}

func (suite *FunctionalTestSuite) SetupSuite() {
	suite.Lock()
	defer suite.Unlock()

	suite._config = config.GetConfig()
	suite.server = factories.ServerFactory()
}

func (suite *FunctionalTestSuite) GetServer() *api.Server {
	return suite.server
}

func (suite *FunctionalTestSuite) TearDownSuite() {
	suite.Lock()
	defer suite.Unlock()

	suite.server.Shutdown(time.Second)
}

func (suite *FunctionalTestSuite) SetupTest() {
	// Make Mock HTTP Server for each URL Block Detector
	_config := config.GetConfig()
	for blockId, blockConfig := range _config.Blocks {
		if blockConfig.Detector.Conditions["url"] != nil {
			successUrl := suite.GetMockHTTPServerURL("Mocked Response OK", http.StatusOK)
			_config.Blocks[blockId].Detector.Conditions["url"] = successUrl
		}
	}
}

func (suite *FunctionalTestSuite) GetMockHTTPServerURL(body string, statusCode int) string {
	suite.Lock()
	defer suite.Unlock()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if body != "" {
			w.Write([]byte(body))
		}
	}))
	suite.httpTestServers = append(suite.httpTestServers, server)

	return server.URL
}

func (suite *FunctionalTestSuite) TearDownTest() {
	suite.Lock()
	defer suite.Unlock()

	for _, server := range suite.httpTestServers {
		server.Close()
	}
	suite.httpTestServers = make([]*httptest.Server, 0)
}
