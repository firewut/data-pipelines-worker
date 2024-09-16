package interfaces

import "bytes"

type StorageLocation interface {
	GetStorage() Storage

	GetBucket() string
	GetLocalDirectory() string
	GetFileName() string

	SetBucket(string)
	SetLocalDirectory(string)
	SetFileName(string)

	GetFilePath() string

	// Storage methods wrappers
	Exists() bool
	Delete() error
	GetObjectBytes() (*bytes.Buffer, error)
	GetStorageName() string
}

type Storage interface {
	// GetStorageName returns the name of the storage
	GetStorageName() string

	// GetStorageDirectory returns the root directory of the storage
	GetStorageDirectory() string
	NewStorageLocation(fileName string) StorageLocation

	// ListObjects returns a list of objects in the given location
	ListObjects(StorageLocation) ([]StorageLocation, error)

	// PutObject copies a file from source to destination
	PutObject(source StorageLocation, destination StorageLocation) (StorageLocation, error)
	// PutObjectBytes copies a file from a buffer to destination
	PutObjectBytes(destination StorageLocation, content *bytes.Buffer) (StorageLocation, error)
	// GetObject
	GetObject(source StorageLocation, destination StorageLocation) error
	// GetObjectBytes returns the content of a file as a buffer
	GetObjectBytes(source StorageLocation) (*bytes.Buffer, error)

	// DeleteObject deletes a file
	DeleteObject(location StorageLocation) error

	// LocationExists checks if a location exists
	LocationExists(location StorageLocation) bool

	// Shutdown closes the storage connection
	Shutdown()
}
