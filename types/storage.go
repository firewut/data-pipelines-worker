package types

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/dataclasses"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type LocalStorage struct {
	name string
	root string
}

func NewLocalStorage(root string) *LocalStorage {
	// _config := config.GetConfig()

	if root == "" {
		root = os.TempDir()

		// TODO: TestNewPipelineBlockDataRegistrySaveOutputLocalStorage fails when this enabled
		// if _config.Storage.Local.RootPath != "" {
		// 	if err := os.MkdirAll(_config.Storage.Local.RootPath, 0755); err != nil {
		// 		panic(err)
		// 	}
		// 	root = _config.Storage.Local.RootPath
		// } else {
		// }
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

func (s *LocalStorage) NewStorageLocation(fileName string) interfaces.StorageLocation {
	return dataclasses.NewStorageLocation(s, s.GetStorageDirectory(), "", fileName)
}

func (s *LocalStorage) ListObjects(location interfaces.StorageLocation) ([]interfaces.StorageLocation, error) {
	logger := config.GetLogger()
	objects := make([]interfaces.StorageLocation, 0)

	var locationDirectory string
	filePath := location.GetFilePath()
	localDirectory := location.GetLocalDirectory()

	if filePath != localDirectory {
		info, err := os.Stat(filePath)
		if err == nil {
			// Ensure it is a directory
			if !info.IsDir() {
				locationDirectory = filepath.Dir(filePath)
			} else {
				locationDirectory = filePath
			}
		}
		if os.IsNotExist(err) {
			locationDirectory = localDirectory
		} else if err != nil {
			return objects, err
		}

	} else {
		locationDirectory = localDirectory
	}

	files, err := os.ReadDir(locationDirectory)
	if err != nil {
		logger.Error(err)
		return objects, err
	}

	for _, file := range files {
		objectStorageLocation := s.NewStorageLocation(file.Name())
		objectStorageLocation.SetLocalDirectory(locationDirectory)
		objects = append(
			objects,
			objectStorageLocation,
		)
	}

	return objects, nil
}

func (s *LocalStorage) PutObject(
	source interfaces.StorageLocation,
	destination interfaces.StorageLocation,
) (interfaces.StorageLocation, error) {
	// Ensure destination directory exists
	if err := os.MkdirAll(destination.GetLocalDirectory(), os.ModePerm); err != nil {
		return s.NewStorageLocation(""), err
	}

	data, err := os.ReadFile(source.GetFilePath())
	if err != nil {
		return s.NewStorageLocation(""), err
	}

	err = os.WriteFile(destination.GetFilePath(), data, 0644)
	if err != nil {
		return s.NewStorageLocation(""), err
	}

	return destination, err
}

func (s *LocalStorage) PutObjectBytes(
	destination interfaces.StorageLocation,
	content *bytes.Buffer,
) (interfaces.StorageLocation, error) {
	mimeType, err := helpers.DetectMimeTypeFromBuffer(*content)
	if err != nil {
		return s.NewStorageLocation(""), err
	}

	// If the file name is empty, generate a new one
	fileName := destination.GetFileName()
	if fileName == "" {
		fileName = uuid.NewString()
	}

	// Replace the file extension with the detected one
	fileName = fileName[:len(fileName)-len(filepath.Ext(fileName))]
	fileName = fmt.Sprintf("%s%s", fileName, mimeType.Extension())

	destinationWithExtension := s.NewStorageLocation(fileName)
	destinationWithExtension.SetLocalDirectory(destination.GetLocalDirectory())

	// Ensure the directory exists
	if err := os.MkdirAll(
		filepath.Dir(
			destinationWithExtension.GetFilePath(),
		),
		os.ModePerm,
	); err != nil {
		return s.NewStorageLocation(""), err
	}

	// Write the file content to the specified path
	file, err := os.Create(destinationWithExtension.GetFilePath())
	if err != nil {
		return s.NewStorageLocation(""), err
	}
	defer file.Close()

	if _, err := content.WriteTo(file); err != nil {
		return s.NewStorageLocation(""), err
	}

	return destinationWithExtension, nil
}

func (s *LocalStorage) GetObject(
	source interfaces.StorageLocation,
	destination interfaces.StorageLocation,
) error {
	return nil
}

func (s *LocalStorage) GetObjectBytes(source interfaces.StorageLocation) (*bytes.Buffer, error) {
	file, err := os.Open(source.GetFilePath())
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
	// TODO: Add sanity checks
	return os.Remove(location.GetFilePath())
}

func (s *LocalStorage) LocationExists(location interfaces.StorageLocation) bool {
	_, err := os.Stat(location.GetFilePath())
	return err == nil
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

func (s *MINIOStorage) NewStorageLocation(fileName string) interfaces.StorageLocation {
	return dataclasses.NewStorageLocation(s, "", s.GetStorageDirectory(), fileName)
}

func (s *MINIOStorage) ListObjects(location interfaces.StorageLocation) ([]interfaces.StorageLocation, error) {
	objects := make([]interfaces.StorageLocation, 0)

	doneCh := make(chan struct{})
	defer close(doneCh)

	for object := range s.Client.ListObjects(
		context.Background(),
		s.GetStorageDirectory(),
		minio.ListObjectsOptions{
			Prefix:    location.GetFileName(),
			Recursive: true,
		},
	) {
		if object.Err != nil {
			return nil, object.Err
		}
		objectStorageLocation := s.NewStorageLocation(object.Key)
		objects = append(objects, objectStorageLocation)
	}

	return objects, nil
}

func (s *MINIOStorage) PutObject(
	source interfaces.StorageLocation,
	destination interfaces.StorageLocation,
) (interfaces.StorageLocation, error) {
	// Get the file extension of Source
	fileName := destination.GetFileName()
	if filepath.Ext(fileName) == "" {
		// Get the file extension of Source from the file content
		content, err := s.localStorage.GetObjectBytes(source)
		if err != nil {
			return s.NewStorageLocation(""), err
		}
		mimeType, err := helpers.DetectMimeTypeFromBuffer(*content)
		if err != nil {
			return s.NewStorageLocation(""), err
		}
		fileName = fmt.Sprintf("%s%s", fileName, mimeType.Extension())
	}

	destinationWithExtension := s.NewStorageLocation(fileName)
	destinationWithExtension.SetBucket(destination.GetBucket())

	_, err := s.Client.FPutObject(
		context.Background(),
		s.GetStorageDirectory(),
		destinationWithExtension.GetFileName(),
		filepath.Join(source.GetLocalDirectory(), source.GetFileName()),
		minio.PutObjectOptions{},
	)
	if err != nil {
		return s.NewStorageLocation(""), err
	}

	return destinationWithExtension, nil
}

func (s *MINIOStorage) PutObjectBytes(
	destination interfaces.StorageLocation,
	content *bytes.Buffer,
) (interfaces.StorageLocation, error) {
	localStorageLocation := s.localStorage.NewStorageLocation(uuid.NewString())
	defer s.localStorage.DeleteObject(localStorageLocation)

	localStorage, err := s.localStorage.PutObjectBytes(localStorageLocation, content)
	if err != nil {
		return s.NewStorageLocation(""), err
	}

	return s.PutObject(localStorage, destination)
}

func (s *MINIOStorage) GetObject(
	source interfaces.StorageLocation,
	destination interfaces.StorageLocation,
) error {
	return s.Client.FGetObject(
		context.Background(),
		s.GetStorageDirectory(),
		source.GetFileName(),
		destination.GetFilePath(),
		minio.GetObjectOptions{},
	)
}

func (s *MINIOStorage) GetObjectBytes(source interfaces.StorageLocation) (*bytes.Buffer, error) {
	localStorageLocation := s.localStorage.NewStorageLocation(uuid.NewString())
	defer s.localStorage.DeleteObject(localStorageLocation)

	err := s.GetObject(source, localStorageLocation)
	if err != nil {
		return nil, err
	}

	return s.localStorage.GetObjectBytes(localStorageLocation)
}

func (s *MINIOStorage) DeleteObject(location interfaces.StorageLocation) error {
	return s.Client.RemoveObject(
		context.Background(),
		s.GetStorageDirectory(),
		location.GetFileName(),
		minio.RemoveObjectOptions{},
	)
}

func (s *MINIOStorage) LocationExists(location interfaces.StorageLocation) bool {
	_, err := s.Client.StatObject(
		context.Background(),
		s.GetStorageDirectory(),
		location.GetFileName(),
		minio.StatObjectOptions{},
	)
	return err == nil
}

func (s *MINIOStorage) Shutdown() {}
