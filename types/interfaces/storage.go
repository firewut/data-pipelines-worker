package interfaces

import "bytes"

// StorageLocation represents a location in a storage
type StorageLocation struct {
	// LocalDirectory is the root directory of the storage
	LocalDirectory string
	// Bucket is the name of the bucket
	Bucket string
	// FileName is the name of the file
	// e.g. <pipeline-slug>/<processing-id>/<block-slug>/output_{i}.<mimetype>
	FileName string
}

type Storage interface {
	// GetStorageName returns the name of the storage
	GetStorageName() string

	// GetStorageDirectory returns the root directory of the storage
	GetStorageDirectory() string
	GetStorageLocation(fileName string) StorageLocation

	// ListObjects returns a list of objects in the given location
	ListObjects(StorageLocation) ([]string, error)

	// PutObject copies a file from source to destination
	PutObject(source StorageLocation, destination StorageLocation) error
	// PutObjectBytes copies a file from a buffer to destination
	PutObjectBytes(destination StorageLocation, content *bytes.Buffer) (StorageLocation, error)
	// GetObject
	GetObject(source StorageLocation, destination StorageLocation) error
	// GetObjectBytes returns the content of a file as a buffer
	GetObjectBytes(source StorageLocation) (*bytes.Buffer, error)

	// DeleteObject deletes a file
	DeleteObject(location StorageLocation) error

	// Shutdown closes the storage connection
	Shutdown()
}
