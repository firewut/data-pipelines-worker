package blocks

type Block interface {
	GetId() string
	GetName() string
	GetDescription() string
	GetSchema() string

	Detect(...interface{}) bool
	Process(interface{}) int
}
