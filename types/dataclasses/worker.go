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

type Worker struct {
	Id     uuid.UUID               `json:"id"`
	Host   string                  `json:"host"`
	IpV4   string                  `json:"ipv4"`
	IpV6   string                  `json:"ipv6"`
	Port   int                     `json:"port"`
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

type WorkerStatus struct {
	Load      float32  `json:"load"`
	Available bool     `json:"available"`
	Version   string   `json:"version"`
	Blocks    []string `json:"blocks"`
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

func (ws *WorkerStatus) GetBlocks() []string {
	return ws.Blocks
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
	if ok && blocks != "" {
		// Split the blocks string by comma and add each block to the worker status.
		workerStatus.Blocks = strings.Split(blocks, ",")
	}

	available, ok := infoFields["available"].(string)
	if ok && (available == "true" || available == "yes") {
		workerStatus.Available = true
	}

	return workerStatus
}
