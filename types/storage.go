package types

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

func (s *LocalStorage) ListObjects(directory string) ([]string, error) {
	return []string{}, fmt.Errorf("not implemented")
}

func (s *LocalStorage) PutObject(directory, localFilePath string, alias string) error {
	return fmt.Errorf("not implemented")
}

func (s *LocalStorage) PutObjectBytes(localDirectory string, fileContent *bytes.Buffer, alias string) (string, error) {
	mimeType, err := DetectMimeTypeFromBuffer(fileContent)
	if err != nil {
		return "", err
	}

	// Put file to localDirectory
	filename := fmt.Sprintf("%s%s", uuid.New(), mimeType.Extension())
	filePath := filepath.Join(localDirectory, filename)

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return "", err
	}

	// Write the file content to the specified path
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := fileContent.WriteTo(file); err != nil {
		return "", err
	}

	return filePath, nil
}

func (s *LocalStorage) GetObject(directory, fileName string, filePath string) (string, error) {
	return "", nil
}

func (s *LocalStorage) GetObjectBytes(directory, fileName string) (*bytes.Buffer, error) {
	file, err := os.Open(filepath.Join(directory, fileName))
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

func (s *MINIOStorage) ListObjects(string) ([]string, error) {
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

func (s *MINIOStorage) PutObject(bucket, source string, alias string) error {
	objectName := filepath.Base(source)

	// Get the file extension
	sourceExt := filepath.Ext(source)

	// If alias is provided, use it as the object name
	if alias != "" {
		objectName = alias
	}

	// If object name does not have an extension, append the source file extension
	if filepath.Ext(objectName) == "" {
		objectName = fmt.Sprintf("%s%s", objectName, sourceExt)
	}

	_, err := s.Client.FPutObject(
		context.Background(),
		s.GetStorageDirectory(),
		objectName,
		source,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *MINIOStorage) PutObjectBytes(bucket string, fileContent *bytes.Buffer, alias string) (string, error) {
	localFilePath, err := s.localStorage.PutObjectBytes("", fileContent, "")
	if err != nil {
		return "", err
	}

	return localFilePath, s.PutObject(s.GetStorageDirectory(), localFilePath, alias)
}

func (s *MINIOStorage) GetObject(bucket, objectName string, filePath string) (string, error) {
	if filePath == "" {
		filePath = filepath.Join(s.localStorage.GetStorageDirectory(), objectName)
	}

	err := s.Client.FGetObject(
		context.Background(),
		s.GetStorageDirectory(),
		objectName,
		filePath,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func (s *MINIOStorage) GetObjectBytes(directory, fileName string) (*bytes.Buffer, error) {
	return nil, fmt.Errorf("not implemented")
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
