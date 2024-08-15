package types

import (
	"strconv"
	"strings"

	"github.com/hashicorp/mdns"
)

type Worker struct {
	Host   string        `json:"host"`
	IpV4   string        `json:"ipv4"`
	IpV6   string        `json:"ipv6"`
	Port   int           `json:"port"`
	Status *WorkerStatus `json:"status"`
}

type WorkerStatus struct {
	Load      float32  `json:"load"`
	Available bool     `json:"available"`
	Version   string   `json:"version"`
	Blocks    []string `json:"blocks"`
}

func NewWorker(entry *mdns.ServiceEntry) *Worker {
	// entry.InfoFields = [version=0.1 load=0.00 available=false blocks=http_request]

	infoFields := make(map[string]interface{})
	for _, field := range entry.InfoFields {
		keyValue := strings.Split(field, "=")
		if len(keyValue) == 2 {
			infoFields[keyValue[0]] = keyValue[1]
		}
	}

	return &Worker{
		Host:   entry.Host,
		Port:   entry.Port,
		IpV4:   entry.AddrV4.String(),
		IpV6:   entry.AddrV6.String(),
		Status: NewWorkerStatus(infoFields),
	}
}

func NewWorkerStatus(infoFields map[string]interface{}) *WorkerStatus {
	workerStatus := &WorkerStatus{
		Load:      0.0,
		Available: false,
		Version:   "",
		Blocks:    make([]string, 0),
	}

	loadStr, ok := infoFields["load"].(string)
	if ok {
		load, _ := strconv.ParseFloat(loadStr, 32)
		workerStatus.Load = float32(load)
	}

	version, ok := infoFields["version"].(string)
	if ok {
		workerStatus.Version = version
	}

	blocks, ok := infoFields["blocks"].(string)
	if ok {
		workerStatus.Blocks = append(workerStatus.Blocks, blocks)
	}

	available, ok := infoFields["available"].(string)
	if ok && (available == "true" || available == "yes") {
		workerStatus.Available = true
	}

	return workerStatus
}
