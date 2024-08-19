package types

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage interface {
	ListObjects(string) ([]string, error)
	PutObject(string, string) error
	GetObject(string, string, string) error
}

type MINIOStorage struct {
	Client *minio.Client
}

func NewMINIOStorage() *MINIOStorage {
	config := GetConfig()
	storageConfig := config.Storage

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
		Client: minioClient,
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

func (s *MINIOStorage) PutObject(bucket, filePath string) error {
	contentType := "application/octet-stream"
	objectName := filepath.Base(filePath)

	_, err := s.Client.FPutObject(
		context.Background(),
		bucket,
		objectName,
		filePath,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *MINIOStorage) GetObject(bucket, objectName string, filePath string) error {
	err := s.Client.FGetObject(
		context.Background(),
		bucket,
		objectName,
		filePath,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}
