package interfaces

type Worker interface {
	GetId() string
	GetHost() string
	GetIPV4() string
	GetIPV6() string
	GetPort() int
	GetStatus() WorkerStatus

	GetAPIEndpoint() string
}

type WorkerStatus interface {
	GetLoad() float32
	GetAvailable() bool
	GetVersion() string
}
