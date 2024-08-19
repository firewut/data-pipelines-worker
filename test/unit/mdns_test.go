package unit_test

import (
	"net"

	"github.com/grandcat/zeroconf"

	"data-pipelines-worker/types"
	"data-pipelines-worker/types/blocks"
)

func (suite *UnitTestSuite) TestNewMDNS() {
	config := types.GetConfig()
	mdnsService := types.NewMDNS(config)

	suite.Equal("data-pipelines-worker", mdnsService.DNSSDStatus.ServiceName)
	suite.Equal("_http._tcp.", mdnsService.DNSSDStatus.ServiceType)
	suite.Equal("local.", mdnsService.DNSSDStatus.ServiceDomain)
	suite.Equal(8080, mdnsService.DNSSDStatus.ServicePort)
	suite.EqualValues(0.0, mdnsService.DNSSDStatus.Load)
	suite.Equal(false, mdnsService.DNSSDStatus.Available)

	suite.Equal([]types.Block{}, mdnsService.GetDetectedBlocks())
	suite.EqualValues(0.0, mdnsService.GetLoad())
	suite.Equal(false, mdnsService.GetAvailable())
}

func (suite *UnitTestSuite) TestMDNSDetectedBlocks() {
	config := types.GetConfig()
	mdnsService := types.NewMDNS(config)

	suite.Equal([]types.Block{}, mdnsService.GetDetectedBlocks())

	detectedBlocks := []types.Block{blocks.NewBlockHTTP()}
	mdnsService.SetDetectedBlocks([]types.Block{
		blocks.NewBlockHTTP(),
	})
	suite.Equal(detectedBlocks, mdnsService.GetDetectedBlocks())
}

func (suite *UnitTestSuite) TestMDNSLoad() {
	config := types.GetConfig()
	mdnsService := types.NewMDNS(config)

	suite.EqualValues(0.0, mdnsService.DNSSDStatus.Load)
	suite.EqualValues(0.0, mdnsService.GetLoad())

	mdnsService.SetLoad(0.5)
	suite.EqualValues(0.5, mdnsService.GetLoad())
}

func (suite *UnitTestSuite) TestMDNSAvailable() {
	config := types.GetConfig()
	mdnsService := types.NewMDNS(config)

	suite.Equal(false, mdnsService.DNSSDStatus.Available)
	suite.Equal(false, mdnsService.GetAvailable())

	mdnsService.SetAvailable(true)
	suite.Equal(true, mdnsService.GetAvailable())
}

func (suite *UnitTestSuite) TestMDNSGetTXT() {
	config := types.GetConfig()
	mdnsService := types.NewMDNS(config)

	txt := mdnsService.GetTXT()
	expected := []string{
		"version=0.1",
		"load=0.00",
		"available=false",
		"blocks=",
	}
	suite.Equal(expected, txt)
}

func (suite *UnitTestSuite) TestGetDiscoveredWorkers() {
	config := types.GetConfig()
	mdnsService := types.NewMDNS(config)

	suite.Equal(len(mdnsService.GetDiscoveredWorkers()), 0)

	discoveredEntries := []*zeroconf.ServiceEntry{
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "localhost",
			},
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.1")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     8080,
		},
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "remotehost",
			},
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.2")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     8080,
		},
	}
	discoveredWorkers := []*types.Worker{
		types.NewWorker(discoveredEntries[0]),
		types.NewWorker(discoveredEntries[1]),
	}

	mdnsService.SetDiscoveredWorkers(discoveredWorkers)

	suite.Equal(mdnsService.GetDiscoveredWorkers(), discoveredWorkers)
}
