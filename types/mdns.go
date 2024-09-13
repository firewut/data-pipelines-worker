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

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
)

type MDNS struct {
	sync.Mutex

	server      *zeroconf.Server
	DNSSDStatus config.DNSSD

	blocksMap map[string]interfaces.Block

	discoverWorkersLock sync.Mutex
	discoveredWorkers   []*dataclasses.Worker
	discoverWorkersDone chan bool

	load      float32
	available bool
}

func NewMDNS() *MDNS {
	config := config.GetConfig()

	return &MDNS{
		DNSSDStatus:         config.DNSSD,
		blocksMap:           make(map[string]interfaces.Block),
		discoveredWorkers:   make([]*dataclasses.Worker, 0),
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

func (m *MDNS) SetBlocks(blocks map[string]interfaces.Block) {
	m.Lock()
	defer m.Unlock()

	m.blocksMap = blocks
}

func (m *MDNS) GetBlocks() map[string]interfaces.Block {
	m.Lock()
	defer m.Unlock()

	return m.blocksMap
}

func (m *MDNS) GetBlock(id string) interfaces.Block {
	m.Lock()
	defer m.Unlock()

	return m.blocksMap[id]
}

func (m *MDNS) GetTXT() []string {
	m.Lock()
	defer m.Unlock()

	keys := make([]string, 0, len(m.blocksMap))
	for k := range m.blocksMap {
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

	config.GetLogger().Debugf(
		"Registering mDNS Service Entry with TXT %s",
		txt,
	)
}

func (m *MDNS) GetDiscoveredWorkers() []*dataclasses.Worker {
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
					workers := make([]*dataclasses.Worker, 0)

					resolver, err := zeroconf.NewResolver(nil)
					if err != nil {
						log.Fatalln("Failed to initialize resolver:", err.Error())
					}
					entries := make(chan *zeroconf.ServiceEntry)
					go func(results <-chan *zeroconf.ServiceEntry) {
						config.GetLogger().Debug("Discovering Workers")
						for entry := range results {
							if !strings.Contains(
								entry.Instance,
								m.DNSSDStatus.ServiceName,
							) {
								continue
							}
							config.GetLogger().Debugf(
								"Discovered Worker %s at %s:%d",
								entry.Instance,
								entry.HostName,
								entry.Port,
							)
							workers = append(workers, dataclasses.NewWorker(entry))
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
						config.GetLogger().Error("Failed to browse:", err.Error())
					}
					<-ctx.Done()
				}(m)

				m.discoverWorkersLock.Unlock()
			}
		}
	}()
}

func (m *MDNS) SetDiscoveredWorkers(workers []*dataclasses.Worker) {
	m.Lock()
	defer m.Unlock()

	m.discoveredWorkers = workers

}

func (m *MDNS) Shutdown() {
	m.Lock()
	defer m.Unlock()

	config.GetLogger().Debugf("Removing mDNS Service Entry")

	if m.server != nil {
		m.server.Shutdown()
		m.discoverWorkersDone <- true
	}
}
