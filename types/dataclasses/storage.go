package dataclasses

import (
	"bytes"
	"path/filepath"
	"sync"

	"data-pipelines-worker/types/interfaces"
)

// StorageLocation represents a location in a storage
type StorageLocation struct {
	sync.Mutex

	Storage interfaces.Storage

	// LocalDirectory is the root directory of the storage
	LocalDirectory string
	// Bucket is the name of the bucket
	Bucket string
	// FileName is the name of the file
	// e.g. <pipeline-slug>/<processing-id>/<block-slug>/output_{i}.<mimetype>
	FileName string
}

func NewStorageLocation(
	storage interfaces.Storage,
	localDirectory,
	bucket,
	fileName string,
) *StorageLocation {
	return &StorageLocation{
		Storage:        storage,
		LocalDirectory: localDirectory,
		Bucket:         bucket,
		FileName:       fileName,
	}
}

func (s *StorageLocation) GetStorage() interfaces.Storage {
	s.Lock()
	defer s.Unlock()

	return s.Storage
}

func (s *StorageLocation) GetBucket() string {
	s.Lock()
	defer s.Unlock()

	return s.Bucket
}

func (s *StorageLocation) GetLocalDirectory() string {
	s.Lock()
	defer s.Unlock()

	return s.LocalDirectory
}

func (s *StorageLocation) GetFileName() string {
	s.Lock()
	defer s.Unlock()

	return s.FileName
}

func (s *StorageLocation) GetFilePath() string {
	return filepath.Join(
		s.GetLocalDirectory(),
		s.GetFileName(),
	)
}

func (s *StorageLocation) SetBucket(bucket string) {
	s.Lock()
	defer s.Unlock()

	s.Bucket = bucket
}

func (s *StorageLocation) SetLocalDirectory(localdirectory string) {
	s.Lock()
	defer s.Unlock()

	s.LocalDirectory = localdirectory
}

func (s *StorageLocation) SetFileName(filename string) {
	s.Lock()
	defer s.Unlock()

	s.FileName = filename
}

func (s *StorageLocation) Exists() bool {
	return s.GetStorage().LocationExists(s)
}

func (s *StorageLocation) Delete() error {
	return s.GetStorage().DeleteObject(s)
}

func (s *StorageLocation) GetObjectBytes() (*bytes.Buffer, error) {
	return s.GetStorage().GetObjectBytes(s)
}

func (s *StorageLocation) GetStorageName() string {
	return s.GetStorage().GetStorageName()
}
