package types

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

type MDNS struct {
	sync.Mutex

	server      *mdns.Server
	DNSSDStatus DNSSD

	detectedBlocksMap map[string]Block
	detectedBlocks    []Block

	discoverWorkersLock sync.Mutex
	discoveredWorkers   []*Worker
	discoverWorkersDone chan bool

	load      float32
	available bool
}

func NewMDNS(config Config) *MDNS {
	return &MDNS{
		DNSSDStatus:         config.DNSSD,
		detectedBlocksMap:   make(map[string]Block),
		detectedBlocks:      make([]Block, 0),
		discoveredWorkers:   make([]*Worker, 0),
		load:                0.0,
		available:           false,
		discoverWorkersDone: make(chan bool, 1),
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

func (m *MDNS) SetDetectedBlocks(detectedBlocks []Block) {
	m.Lock()
	defer m.Unlock()

	m.detectedBlocks = detectedBlocks
	for _, block := range detectedBlocks {
		m.detectedBlocksMap[block.GetId()] = block
	}
}

func (m *MDNS) GetDetectedBlocks() []Block {
	m.Lock()
	defer m.Unlock()

	return m.detectedBlocks
}

func (m *MDNS) GetDetectedBlock(id string) Block {
	m.Lock()
	defer m.Unlock()

	return m.detectedBlocksMap[id]
}

func (m *MDNS) GetTXT() []string {
	m.Lock()
	defer m.Unlock()

	keys := make([]string, 0, len(m.detectedBlocks))
	for k := range m.detectedBlocksMap {
		keys = append(keys, k)
	}

	return []string{
		fmt.Sprintf("version=%s", m.DNSSDStatus.Version),
		fmt.Sprintf("load=%.2f", m.load),
		fmt.Sprintf("available=%t", m.available),
		fmt.Sprintf(
			"blocks=%s",
			strings.Join(
				keys,
				",",
			),
		),
	}
}

func (m *MDNS) Announce() {
	m.SetAvailable(true)

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
		"Registering mDNS Service Entry with TXT %s",
		txt,
	)

	if server, err := mdns.NewServer(&mdns.Config{Zone: service}); err == nil {
		m.server = server
	} else {
		panic(err)
	}
}

func (m *MDNS) GetDiscoveredWorkers() []*Worker {
	m.Lock()
	defer m.Unlock()

	return m.discoveredWorkers
}

func (m *MDNS) DiscoverWorkers() {
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-m.discoverWorkersDone:
				ticker.Stop()
				close(m.discoverWorkersDone)
				return
			case <-ticker.C:
				m.discoverWorkersLock.Lock()

				workers := make([]*Worker, 0)
				entriesCh := make(chan *mdns.ServiceEntry)

				go func(_m *MDNS, _workers []*Worker) {
					for entry := range entriesCh {
						if !strings.Contains(entry.Name, m.DNSSDStatus.ServiceName) {
							continue
						}
						_workers = append(_workers, NewWorker(entry))
					}
					if len(_workers) > 0 {
						_m.SetDiscoveredWorkers(_workers)
					}
				}(m, workers)

				params := &mdns.QueryParam{
					Service:     m.DNSSDStatus.ServiceType,
					Domain:      m.DNSSDStatus.ServiceDomain,
					Timeout:     5 * time.Second,
					Entries:     entriesCh,
					DisableIPv6: true,
				}

				// Add check for Filter and ServiceName herez
				if err := mdns.Query(params); err != nil {
					GetLogger().Errorf("Failed to query mDNS: %s", err)
				}
				close(entriesCh)

				m.discoverWorkersLock.Unlock()
			}
		}
	}()
}

func (m *MDNS) SetDiscoveredWorkers(workers []*Worker) {
	m.Lock()
	defer m.Unlock()

	m.discoveredWorkers = workers
}

func (m *MDNS) Shutdown() {
	m.Lock()
	defer m.Unlock()

	GetLogger().Debugf("Removing mDNS Service Entry")

	if m.server != nil {
		m.server.Shutdown()
		m.discoverWorkersDone <- true
	}
}
