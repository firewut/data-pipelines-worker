package factories

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grandcat/zeroconf"
	"github.com/sashabaranov/go-openai"

	"data-pipelines-worker/api"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/interfaces"
)

func NewServer() *api.Server {
	server := api.NewServer()
	server.SetPort(0)
	return server
}

func NewServerWithHandlers(configs ...config.Config) *api.Server {
	server := api.NewServer(configs...)
	server.SetPort(0)
	server.SetAPIMiddlewares()
	server.SetAPIHandlers()
	return server
}

func NewWorkerServerWithHandlers(ctx context.Context, available bool, configs ...config.Config) (*api.Server, interfaces.Worker, error) {
	server := NewServerWithHandlers(configs...)
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

func NewOpenAIClient(url string) *openai.Client {
	return openai.NewClientWithConfig(
		openai.ClientConfig{
			BaseURL:            url,
			APIType:            openai.APITypeOpenAI,
			AssistantVersion:   "v2",
			OrgID:              "",
			HTTPClient:         &http.Client{},
			EmptyMessagesLimit: 0,
		},
	)
}

func NewTelegramClient(url string) (*tgbotapi.BotAPI, error) {
	apiEndpoint := url + "/bot%s/%s"
	return tgbotapi.NewBotAPIWithAPIEndpoint("TOKEN", apiEndpoint)
}

func GetPNGImageBuffer(width int, height int) bytes.Buffer {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill it with white color
	_color := color.RGBA{100, 100, 100, 100}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, _color)
		}
	}

	// Draw lines every 50 pixels (vertical and horizontal)
	lineColor := color.RGBA{0, 0, 0, 255} // Black color for lines

	// Draw vertical lines
	for x := 0; x < width; x += 50 {
		for y := 0; y < height; y++ {
			img.Set(x, y, lineColor)
		}
	}

	// Draw horizontal lines
	for y := 0; y < height; y += 50 {
		for x := 0; x < width; x++ {
			img.Set(x, y, lineColor)
		}
	}

	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		panic(err)
	}

	return *buf
}
