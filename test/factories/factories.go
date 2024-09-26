package factories

import (
	"context"
	"fmt"
	"net"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"

	"data-pipelines-worker/api"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
)

func NewServer() *api.Server {
	server := api.NewServer()
	server.SetPort(0)
	return server
}

func NewServerWithHandlers() *api.Server {
	server := api.NewServer()
	server.SetPort(0)
	server.SetAPIMiddlewares()
	server.SetAPIHandlers()
	return server
}

func NewWorkerServerWithHandlers(ctx context.Context, available bool) (*api.Server, interfaces.Worker, error) {
	server := NewServerWithHandlers()
	go server.Start(ctx)

	<-server.Ready

	// This one for testing purposes. Sometimes server needs more time to Listen in test
	time.Sleep(time.Millisecond)

	u, err := url.Parse(server.GetAPIAddress())
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return server, nil, err
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		fmt.Println("Error converting port to integer:", err)
		return server, nil, err
	}

	worker := dataclasses.NewWorker(
		&zeroconf.ServiceEntry{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "remotehost",
			},
			HostName: u.Hostname(),
			AddrIPv4: []net.IP{net.ParseIP("localhost")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     port,
			Text:     server.GetMDNS().GetTXT(),
		},
	)

	return server, worker, err
}

func NewWorkerServer(
	mockServerFunction func(
		string,
		int,
		time.Duration,
		map[string]string,
	) *httptest.Server,
	statusCode int,
	workerAvailable bool,
	blockIds []string,
	responseDelay time.Duration,
	bodyMapping map[string]string,
) (*httptest.Server, interfaces.Worker, error) {
	server := mockServerFunction(
		"",
		statusCode,
		responseDelay,
		bodyMapping,
	)
	workerEntry, err := NewWorkerRegistryEntry(
		server, workerAvailable, blockIds,
	)

	return server, workerEntry, err
}

func NewWorkerRegistryEntry(
	workerServer *httptest.Server,
	available bool,
	blockIds []string,
) (interfaces.Worker, error) {
	u, err := url.Parse(workerServer.URL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return nil, err
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		fmt.Println("Error converting port to integer:", err)
		return nil, err
	}

	return dataclasses.NewWorker(
		&zeroconf.ServiceEntry{
			ServiceRecord: zeroconf.ServiceRecord{
				Instance: "remotehost",
			},
			HostName: u.Hostname(),
			AddrIPv4: []net.IP{net.ParseIP("192.168.1.2")},
			AddrIPv6: []net.IP{net.ParseIP("::1")},
			Port:     port,
			Text: []string{
				"version=0.1",
				"load=0.00",
				fmt.Sprintf("available=%t", available),
				fmt.Sprintf(
					"blocks=block_http,%s,block_gpu_image_resize",
					strings.Join(blockIds, ","),
				),
			},
		},
	), nil
}
