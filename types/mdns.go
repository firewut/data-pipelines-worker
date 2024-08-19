package types

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"
)

type MDNS struct {
	sync.Mutex

	server      *zeroconf.Server
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

	server, err := zeroconf.Register(
		fmt.Sprintf(
			"%s-%s",
			m.DNSSDStatus.ServiceName,
			uuid.New().String(),
		),
		m.DNSSDStatus.ServiceType,
		m.DNSSDStatus.ServiceDomain,
		m.DNSSDStatus.ServicePort,
		txt,
		nil,
	)
	if err != nil {
		panic(err)
	}
	m.server = server

	GetLogger().Debugf(
		"Registering mDNS Service Entry with TXT %s",
		txt,
	)
}

func (m *MDNS) GetDiscoveredWorkers() []*Worker {
	m.Lock()
	defer m.Unlock()

	return m.discoveredWorkers
}

func (m *MDNS) DiscoverWorkers() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-m.discoverWorkersDone:
				ticker.Stop()
				close(m.discoverWorkersDone)
				return
			case <-ticker.C:
				m.discoverWorkersLock.Lock()

				go func(_m *MDNS) {
					workers := make([]*Worker, 0)

					resolver, err := zeroconf.NewResolver(nil)
					if err != nil {
						log.Fatalln("Failed to initialize resolver:", err.Error())
					}
					entries := make(chan *zeroconf.ServiceEntry)
					go func(results <-chan *zeroconf.ServiceEntry) {
						GetLogger().Debug("Discovering Workers")
						for entry := range results {
							if !strings.Contains(
								entry.Instance,
								m.DNSSDStatus.ServiceName,
							) {
								continue
							}
							GetLogger().Debugf(
								"Discovered Worker %s at %s:%d",
								entry.Instance,
								entry.HostName,
								entry.Port,
							)
							workers = append(workers, NewWorker(entry))
						}
						_m.SetDiscoveredWorkers(workers)
					}(entries)

					ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
					defer cancel()
					err = resolver.Browse(
						ctx,
						m.DNSSDStatus.ServiceType,
						m.DNSSDStatus.ServiceDomain,
						entries,
					)
					if err != nil {
						GetLogger().Error("Failed to browse:", err.Error())
					}
					<-ctx.Done()
				}(m)

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
