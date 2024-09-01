package interfaces

import "bytes"

type Storage interface {
	ListObjects(location string) ([]string, error)

	PutObject(destination string, source string) error
	PutObjectBytes(destination string, content *bytes.Buffer) (string, error)

	GetObject(string, string, string) (string, error)
	GetObjectBytes(string, string) (*bytes.Buffer, error)
}
