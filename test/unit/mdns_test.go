package unit_test

import (
	"data-pipelines-worker/types"
	"data-pipelines-worker/types/blocks"
)

func (suite *UnitTestSuite) TestNewMDNS() {
	config := types.NewConfig()
	mdns := types.NewMDNS(config)

	suite.Equal("data-pipelines-worker", mdns.DNSSDStatus.ServiceName)
	suite.Equal("_http._tcp.", mdns.DNSSDStatus.ServiceType)
	suite.Equal("local.", mdns.DNSSDStatus.ServiceDomain)
	suite.Equal(8080, mdns.DNSSDStatus.ServicePort)
	suite.Equal("", mdns.DNSSDStatus.Load)
	suite.Equal(false, mdns.DNSSDStatus.Available)

	suite.Equal([]types.Block{}, mdns.GetDetectedBlocks())
	suite.EqualValues(0.0, mdns.GetLoad())
	suite.Equal(false, mdns.GetAvailable())
}

func (suite *UnitTestSuite) TestMDNSDetectedBlocks() {
	config := types.NewConfig()
	mdns := types.NewMDNS(config)

	suite.Equal([]types.Block{}, mdns.GetDetectedBlocks())

	detectedBlocks := []types.Block{blocks.NewBlockHTTP()}
	mdns.SetDetectedBlocks([]types.Block{
		blocks.NewBlockHTTP(),
	})
	suite.Equal(detectedBlocks, mdns.GetDetectedBlocks())
}

func (suite *UnitTestSuite) TestMDNSLoad() {
	config := types.NewConfig()
	mdns := types.NewMDNS(config)

	suite.Equal("", mdns.DNSSDStatus.Load)
	suite.EqualValues(0.0, mdns.GetLoad())

	mdns.SetLoad(0.5)
	suite.EqualValues(0.5, mdns.GetLoad())
}

func (suite *UnitTestSuite) TestMDNSAvailable() {
	config := types.NewConfig()
	mdns := types.NewMDNS(config)

	suite.Equal(false, mdns.DNSSDStatus.Available)
	suite.Equal(false, mdns.GetAvailable())

	mdns.SetAvailable(true)
	suite.Equal(true, mdns.GetAvailable())
}

func (suite *UnitTestSuite) TestMDNSGetTXT() {
	config := types.NewConfig()
	mdns := types.NewMDNS(config)

	txt := mdns.GetTXT()
	expected := []string{
		"version=0.1",
		"load=0.00",
		"available=false",
		"blocks=",
	}
	suite.Equal(expected, txt)
}
