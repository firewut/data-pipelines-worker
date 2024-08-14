package types

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/mdns"
)

type MDNS struct {
	sync.Mutex

	server         *mdns.Server
	DNSSDStatus    DNSSD
	detectedBlocks []string
	load           float32
	available      bool
}

func NewMDNS(config Config) *MDNS {
	return &MDNS{
		DNSSDStatus:    config.DNSSD,
		detectedBlocks: make([]string, 0),
		load:           0.0,
		available:      false,
	}
}

func (m *MDNS) SetLoad(load float32) {
	m.Lock()
	defer m.Unlock()

	m.load = load
}

func (m *MDNS) SetAvailable(available bool) {
	m.Lock()
	defer m.Unlock()

	m.available = available
}

func (m *MDNS) GetLoad() float32 {
	m.Lock()
	defer m.Unlock()

	return m.load
}

func (m *MDNS) GetAvailable() bool {
	m.Lock()
	defer m.Unlock()

	return m.available
}

func (m *MDNS) SetDetectedBlocks(detectedBlocks []string) {
	m.Lock()
	defer m.Unlock()

	m.detectedBlocks = detectedBlocks
}

func (m *MDNS) GetDetectedBlocks() []string {
	m.Lock()
	defer m.Unlock()

	return m.detectedBlocks
}

func (m *MDNS) GetTXT() []string {
	m.Lock()
	defer m.Unlock()

	return []string{
		fmt.Sprintf("version=%s", m.DNSSDStatus.Version),
		fmt.Sprintf("load=%.2f", m.load),
		fmt.Sprintf("available=%t", m.available),
		fmt.Sprintf("blocks=%s", strings.Join(m.detectedBlocks, ",")),
	}
}

func (m *MDNS) Advertise() {
	txt := m.GetTXT()

	m.Lock()
	defer m.Unlock()

	service, err := mdns.NewMDNSService(
		m.DNSSDStatus.ServiceName,
		m.DNSSDStatus.ServiceType,
		m.DNSSDStatus.ServiceDomain,
		"",
		m.DNSSDStatus.ServicePort,
		nil,
		txt,
	)
	if err != nil {
		panic(err)
	}

	GetLogger().Debugf(
		"Starting mDNS server %s with TXT %s",
		service,
		txt,
	)

	if server, err := mdns.NewServer(&mdns.Config{Zone: service}); err == nil {
		m.server = server
	} else {
		panic(err)
	}
}

func (m *MDNS) Shutdown() {
	m.Lock()
	defer m.Unlock()

	GetLogger().Debugf(
		"Shutting down mDNS server %s",
		m.server,
	)

	if m.server != nil {
		m.server.Shutdown()
	}
}
