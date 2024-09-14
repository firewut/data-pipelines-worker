package interfaces

import "bytes"

type Storage interface {
	ListObjects(location string) ([]string, error)

	PutObject(destination string, source string, alias string) error
	PutObjectBytes(destination string, content *bytes.Buffer, alias string) (string, error)

	GetObject(string, string, string) (string, error)
	GetObjectBytes(string, string) (*bytes.Buffer, error)

	GetStorageDirectory() string

	Shutdown()
}
