package unit_test

import (
	"net"

	"github.com/grandcat/zeroconf"

	"data-pipelines-worker/types"
	"data-pipelines-worker/types/dataclasses"
)

func (suite *UnitTestSuite) TestNewMDNS() {
	mdnsService := types.NewMDNS()

	suite.Equal("data-pipelines-worker", mdnsService.DNSSDStatus.ServiceName)
	suite.Equal("_http._tcp.", mdnsService.DNSSDStatus.ServiceType)
	suite.Equal("local.", mdnsService.DNSSDStatus.ServiceDomain)
	suite.Equal(8080, mdnsService.DNSSDStatus.ServicePort)
	suite.EqualValues(0.0, mdnsService.DNSSDStatus.Load)
	suite.Equal(false, mdnsService.DNSSDStatus.Available)

	suite.EqualValues(0.0, mdnsService.GetLoad())
	suite.Equal(false, mdnsService.GetAvailable())
}

func (suite *UnitTestSuite) TestMDNSLoad() {
	mdnsService := types.NewMDNS()

	suite.EqualValues(0.0, mdnsService.DNSSDStatus.Load)
	suite.EqualValues(0.0, mdnsService.GetLoad())

	mdnsService.SetLoad(0.5)
	suite.EqualValues(0.5, mdnsService.GetLoad())
}

func (suite *UnitTestSuite) TestMDNSAvailable() {
	mdnsService := types.NewMDNS()

	suite.Equal(false, mdnsService.DNSSDStatus.Available)
	suite.Equal(false, mdnsService.GetAvailable())

	mdnsService.SetAvailable(true)
	suite.Equal(true, mdnsService.GetAvailable())
}

func (suite *UnitTestSuite) TestMDNSGetTXT() {
	mdnsService := types.NewMDNS()

	txt := mdnsService.GetTXT()
	expected := []string{
		"version=0.1",
		"load=0.00",
		"available=false",
	}
	suite.Equal(expected, txt)
}

func (suite *UnitTestSuite) TestGetDiscoveredWorkers() {
	mdnsService := types.NewMDNS()

	suite.Equal(len(mdnsService.GetDiscoveredWorkers()), 0)

	discoveredEntries := []*zeroconf.ServiceEntry{
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "localhost",
			},
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.1")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     8080,
			Text:     []string{"version=0.1", "load=0.00", "available=false"},
		},
		{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "remotehost",
			},
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.2")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     8080,
			Text:     []string{"version=0.1", "load=0.00", "available=true"},
		},
	}
	discoveredWorkers := []*dataclasses.Worker{
		dataclasses.NewWorker(discoveredEntries[0]),
		dataclasses.NewWorker(discoveredEntries[1]),
	}

	mdnsService.SetDiscoveredWorkers(discoveredWorkers)

	suite.Equal(mdnsService.GetDiscoveredWorkers(), discoveredWorkers)
}
