package dataclasses

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"

	"data-pipelines-worker/types/interfaces"
)

// Worker represents the structure of a worker in the system. It includes
// details like the worker's host, IP addresses, port, and status information.
//
// swagger:model
type Worker struct { //in `dataclasses` package
	// The unique identifier for the worker
	// example: "d9b2d63d-5f23-e4d7-6b7f-3f2f25d93a7a"
	Id uuid.UUID `json:"id"`

	// The hostname of the worker
	// example: "worker-hostname.local"
	Host string `json:"host"`

	// The IPv4 address of the worker
	// example: "192.168.1.10"
	IpV4 string `json:"ipv4"`

	// The IPv6 address of the worker
	// example: "fe80::a00:27ff:fe9e:3c28"
	IpV6 string `json:"ipv6"`

	// The port on which the worker is running
	// example: 8080
	Port int `json:"port"`

	// The current status of the worker
	// example: {"load": 0.5, "available": true, "version": "1.0.0"}
	Status interfaces.WorkerStatus `json:"status"`
}

func (w *Worker) GetId() string {
	return w.Id.String()
}

func (w *Worker) GetHost() string {
	return w.Host
}

func (w *Worker) GetIPV4() string {
	return w.IpV4
}

func (w *Worker) GetIPV6() string {
	return w.IpV6
}

func (w *Worker) GetPort() int {
	return w.Port
}

func (w *Worker) GetStatus() interfaces.WorkerStatus {
	return w.Status
}

func (w *Worker) GetAPIEndpoint() string {
	host := w.GetHost()
	if host == "" {
		host = w.GetIPV4()
	}
	if host == "" {
		host = w.GetIPV6()
	}

	return fmt.Sprintf(
		"http://%s",
		net.JoinHostPort(host, strconv.Itoa(w.Port)),
	)
}

// WorkerStatus represents the status information of a worker. It includes
// the worker's load, availability, and version information.
//
// swagger:model
type WorkerStatus struct {
	// The load on the worker, as a floating-point value
	// example: 0.75
	Load float32 `json:"load"`

	// Whether the worker is available or not
	// example: true
	Available bool `json:"available"`

	// The version of the worker's software
	// example: "v1.2.3"
	Version string `json:"version"`
}

func (ws *WorkerStatus) GetLoad() float32 {
	return ws.Load
}

func (ws *WorkerStatus) GetAvailable() bool {
	return ws.Available
}

func (ws *WorkerStatus) GetVersion() string {
	return ws.Version
}

func NewWorker(entry *zeroconf.ServiceEntry) *Worker {
	infoFields := make(map[string]interface{})
	for _, field := range entry.Text {
		keyValue := strings.Split(field, "=")
		if len(keyValue) == 2 {
			infoFields[keyValue[0]] = keyValue[1]
		}
	}

	return &Worker{
		Id:     uuid.New(),
		Host:   entry.HostName,
		Port:   entry.Port,
		IpV4:   entry.AddrIPv4[0].String(),
		IpV6:   entry.AddrIPv6[0].String(),
		Status: NewWorkerStatus(infoFields),
	}
}

func NewWorkerStatus(infoFields map[string]interface{}) interfaces.WorkerStatus {
	workerStatus := &WorkerStatus{
		Load:      0.0,
		Available: false,
		Version:   "",
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

	available, ok := infoFields["available"].(string)
	if ok && (available == "true" || available == "yes") {
		workerStatus.Available = true
	}

	return workerStatus
}
