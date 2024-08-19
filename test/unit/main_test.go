package unit_test

import (
	"sync"
	"testing"

	"data-pipelines-worker/types"

	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	suite.Suite
	sync.RWMutex

	config types.Config
}

func TestUnitTestSuite(t *testing.T) {
	// Set Logger level to debug
	types.GetLogger().SetLevel(log.INFO)

	suite.Run(t, new(UnitTestSuite))
}

func (suite *UnitTestSuite) SetupSuite() {
	suite.Lock()
	defer suite.Unlock()

	suite.config = types.GetConfig()
}

func (suite *UnitTestSuite) TearDownSuite() {
}

func (suite *UnitTestSuite) SetupTest() {

}

func (suite *UnitTestSuite) TearDownTest() {
}
