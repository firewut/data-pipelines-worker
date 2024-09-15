package types

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

type LocalStorage struct {
	name string
	root string
}

func NewLocalStorage(root string) *LocalStorage {
	_config := config.GetConfig()

	if root == "" {
		if _config.Storage.Local.RootPath != "" {
			if err := os.MkdirAll(_config.Storage.Local.RootPath, 0755); err != nil {
				panic(err)
			}
			root = _config.Storage.Local.RootPath
		} else {
			root = os.TempDir()
		}
	}
	return &LocalStorage{
		name: "local",
		root: root,
	}
}

func (s *LocalStorage) GetStorageName() string {
	return s.name
}

func (s *LocalStorage) GetStorageDirectory() string {
	return s.root
}

func (s *LocalStorage) GetStorageLocation(fileName string) interfaces.StorageLocation {
	return interfaces.StorageLocation{
		LocalDirectory: s.GetStorageDirectory(),
		FileName:       fileName,
	}
}

func (s *LocalStorage) ListObjects(location interfaces.StorageLocation) ([]string, error) {
	objects := []string{}

	files, err := os.ReadDir(location.LocalDirectory)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		objects = append(
			objects,
			filepath.Join(
				location.LocalDirectory,
				file.Name(),
			),
		)
	}

	return objects, nil
}

func (s *LocalStorage) PutObject(
	source interfaces.StorageLocation,
	destination interfaces.StorageLocation,
) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(destination.LocalDirectory, os.ModePerm); err != nil {
		return err
	}

	data, err := os.ReadFile(filepath.Join(source.LocalDirectory, source.FileName))
	if err != nil {
		fmt.Println("Error reading file:", err)
		return err
	}

	err = os.WriteFile(filepath.Join(destination.LocalDirectory, destination.FileName), data, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return err
	}

	return nil
}

func (s *LocalStorage) PutObjectBytes(
	destination interfaces.StorageLocation,
	content *bytes.Buffer,
) (
	interfaces.StorageLocation,
	error,
) {
	// PutObjectBytes(localDirectory string, fileContent *bytes.Buffer, alias string)
	mimeType, err := DetectMimeTypeFromBuffer(content)
	if err != nil {
		return interfaces.StorageLocation{}, err
	}

	if destination.FileName == "" {
		destination.FileName = uuid.NewString()
	}

	// Replace the file extension with the detected one
	destination.FileName = destination.FileName[:len(destination.FileName)-len(filepath.Ext(destination.FileName))]
	destination.FileName = fmt.Sprintf("%s%s", destination.FileName, mimeType.Extension())

	// Ensure the directory exists
	if err := os.MkdirAll(destination.LocalDirectory, os.ModePerm); err != nil {
		return interfaces.StorageLocation{}, err
	}

	localFile := filepath.Join(destination.LocalDirectory, destination.FileName)
	// Write the file content to the specified path
	file, err := os.Create(localFile)
	if err != nil {
		return interfaces.StorageLocation{}, err
	}
	defer file.Close()

	if _, err := content.WriteTo(file); err != nil {
		return interfaces.StorageLocation{}, err
	}

	return interfaces.StorageLocation{
		LocalDirectory: destination.LocalDirectory,
		FileName:       destination.FileName,
	}, nil
}

func (s *LocalStorage) GetObject(
	source interfaces.StorageLocation,
	destination interfaces.StorageLocation,
) error {
	return nil
}

func (s *LocalStorage) GetObjectBytes(source interfaces.StorageLocation) (*bytes.Buffer, error) {
	file, err := os.Open(
		filepath.Join(source.LocalDirectory, source.FileName),
	)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

func (s *LocalStorage) DeleteObject(location interfaces.StorageLocation) error {
	return os.Remove(
		filepath.Join(location.LocalDirectory, location.FileName),
	)
}

func (s *LocalStorage) Shutdown() {}

type MINIOStorage struct {
	Client *minio.Client

	name         string
	bucket       string
	localStorage interfaces.Storage // Some operations requires local storage
}

func NewMINIOStorage() *MINIOStorage {
	storageConfig := config.GetConfig().Storage

	minioClient, err := minio.New(
		storageConfig.Minio.Url,
		&minio.Options{
			Creds:  credentials.NewStaticV4(storageConfig.Minio.AccessKey, storageConfig.Minio.SecretKey, ""),
			Secure: false, // Set to true if using HTTPS
		},
	)
	if err != nil {
		panic(err)
	}

	return &MINIOStorage{
		Client:       minioClient,
		bucket:       storageConfig.Minio.Bucket,
		localStorage: NewLocalStorage(""),
		name:         "minio",
	}
}

func (s *MINIOStorage) GetStorageName() string {
	return s.name
}

func (s *MINIOStorage) GetStorageDirectory() string {
	return s.bucket
}

func (s *MINIOStorage) GetStorageLocation(fileName string) interfaces.StorageLocation {
	return interfaces.StorageLocation{
		Bucket:   s.GetStorageDirectory(),
		FileName: fileName,
	}
}

func (s *MINIOStorage) ListObjects(location interfaces.StorageLocation) ([]string, error) {
	objects := []string{}

	doneCh := make(chan struct{})
	defer close(doneCh)

	for object := range s.Client.ListObjects(
		context.Background(),
		s.GetStorageDirectory(),
		minio.ListObjectsOptions{
			Recursive: true,
		},
	) {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

func (s *MINIOStorage) PutObject(
	source interfaces.StorageLocation,
	destination interfaces.StorageLocation,
) error {
	// Get the file extension of Source
	if filepath.Ext(destination.FileName) == "" {
		// Get the file extension of Source from the file content
		content, err := s.localStorage.GetObjectBytes(source)
		if err != nil {
			return err
		}
		mimeType, err := DetectMimeTypeFromBuffer(content)
		if err != nil {
			return err
		}
		destination.FileName = fmt.Sprintf("%s%s", destination.FileName, mimeType.Extension())
	}

	_, err := s.Client.FPutObject(
		context.Background(),
		s.GetStorageDirectory(),
		destination.FileName,
		filepath.Join(source.LocalDirectory, source.FileName),
		minio.PutObjectOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *MINIOStorage) PutObjectBytes(
	destination interfaces.StorageLocation,
	content *bytes.Buffer,
) (
	interfaces.StorageLocation,
	error,
) {
	// Save the file to the local storage
	localStorage, err := s.localStorage.PutObjectBytes(
		interfaces.StorageLocation{
			LocalDirectory: s.localStorage.GetStorageDirectory(),
			FileName:       uuid.NewString(),
		},
		content,
	)
	if err != nil {
		return interfaces.StorageLocation{}, err
	}

	return localStorage, s.PutObject(localStorage, destination)
}

func (s *MINIOStorage) GetObject(
	source interfaces.StorageLocation,
	destination interfaces.StorageLocation,
) error {
	return s.Client.FGetObject(
		context.Background(),
		s.GetStorageDirectory(),
		source.FileName,
		filepath.Join(destination.LocalDirectory, destination.FileName),
		minio.GetObjectOptions{},
	)
}

func (s *MINIOStorage) GetObjectBytes(source interfaces.StorageLocation) (*bytes.Buffer, error) {
	destination := interfaces.StorageLocation{
		LocalDirectory: s.localStorage.GetStorageDirectory(),
		FileName:       uuid.NewString(),
	}
	err := s.GetObject(source, destination)
	if err != nil {
		return nil, err
	}

	return s.localStorage.GetObjectBytes(destination)
}

func (s *MINIOStorage) DeleteObject(location interfaces.StorageLocation) error {
	return s.Client.RemoveObject(
		context.Background(),
		s.GetStorageDirectory(),
		location.FileName,
		minio.RemoveObjectOptions{},
	)
}

func (s *MINIOStorage) Shutdown() {}

func DetectMimeTypeFromBuffer(largeBuffer *bytes.Buffer) (*mimetype.MIME, error) {
	var detectorBytesLen int64 = 261

	// Create a new buffer to hold the copied data
	var copiedBuffer bytes.Buffer

	// Create a TeeReader that reads from largeBuffer and writes to copiedBuffer
	teeReader := io.TeeReader(io.LimitReader(largeBuffer, detectorBytesLen), &copiedBuffer)

	smallBuffer := make([]byte, detectorBytesLen)
	bytesRead, err := io.ReadFull(teeReader, smallBuffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	largeBuffer.Reset()
	largeBuffer.Write(copiedBuffer.Bytes())

	return mimetype.Detect(smallBuffer[:bytesRead]), nil
}
