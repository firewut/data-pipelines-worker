package types

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

type LocalStorage struct{}

func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

func (s *LocalStorage) ListObjects(directory string) ([]string, error) {
	return []string{}, fmt.Errorf("not implemented")
}

func (s *LocalStorage) PutObject(directory, localFilePath string) error {
	return fmt.Errorf("not implemented")
}

func (s *LocalStorage) PutObjectBytes(localDirectory string, fileContent *bytes.Buffer) (string, error) {
	mimeType, err := DetectMimeTypeFromBuffer(fileContent)
	if err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("output_*%s", mimeType.Extension()))
	if err != nil {
		config.GetLogger().Errorf("Failed to create temporary file: %v", err)
		return "", err
	}
	defer tmpFile.Close()

	_, err = tmpFile.Write(fileContent.Bytes())
	if err != nil {
		config.GetLogger().Errorf("Failed to write to temporary file: %v", err)
		return "", err
	}
	return tmpFile.Name(), nil
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

type MINIOStorage struct {
	Client       *minio.Client
	localStorage interfaces.Storage // Some operations requires local storage
}

func NewMINIOStorage() *MINIOStorage {
	storageConfig := config.GetConfig().Storage

	minioClient, err := minio.New(
		storageConfig.Url,
		&minio.Options{
			Creds:  credentials.NewStaticV4(storageConfig.AccessKey, storageConfig.SecretKey, ""),
			Secure: false, // Set to true if using HTTPS
		},
	)
	if err != nil {
		fmt.Println("Error creating MinIO client:", err)
	}

	return &MINIOStorage{
		Client:       minioClient,
		localStorage: NewLocalStorage(),
	}
}

func (s *MINIOStorage) ListObjects(bucket string) ([]string, error) {
	objects := []string{}

	doneCh := make(chan struct{})
	defer close(doneCh)

	for object := range s.Client.ListObjects(
		context.Background(),
		bucket,
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

func (s *MINIOStorage) PutObject(bucket, source string) error {
	objectName := filepath.Base(source)

	_, err := s.Client.FPutObject(
		context.Background(),
		bucket,
		objectName,
		source,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *MINIOStorage) PutObjectBytes(bucket string, fileContent *bytes.Buffer) (string, error) {
	localFilePath, err := s.localStorage.PutObjectBytes("", fileContent)
	if err != nil {
		return "", err
	}

	return localFilePath, s.PutObject(bucket, localFilePath)
}

func (s *MINIOStorage) GetObject(bucket, objectName string, filePath string) (string, error) {
	err := s.Client.FGetObject(
		context.Background(),
		bucket,
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
