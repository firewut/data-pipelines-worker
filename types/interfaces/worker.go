package interfaces

type Worker interface {
	GetHost() string
	GetIPV4() string
	GetIPV6() string
	GetPort() int
	GetStatus() WorkerStatus
}

type WorkerStatus interface {
	GetLoad() float32
	GetAvailable() bool
	GetVersion() string
	GetBlocks() []string
}
