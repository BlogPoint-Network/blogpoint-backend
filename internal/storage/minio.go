package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"log"
	"os"
)

var MinioClient *minio.Client

const bucketName = "blogpoint-bucket"

// InitMinio инициализирует MinIO клиент и создает бакет, если его нет
func InitMinio() {
	var err error
	MinioClient, err = minio.New(
		os.Getenv("MINIO_INTERNAL_ENDPOINT"),
		&minio.Options{
			Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), ""),
			Secure: false, // false, если MinIO работает без SSL
		})
	if err != nil {
		log.Fatalf("MinIO initialization error: %v", err)
	}

	// Проверяем существование бакета
	exists, err := MinioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		log.Fatalf("Bucket check error: %v", err)
	}

	// Создаем бакет, если его нет
	if !exists {
		err = MinioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Error creating bucket: %v", err)
		}
		fmt.Printf("Бакет %s создан\n", bucketName)

		policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::` + bucketName + `/*"
			}
		]
	}`

		err = MinioClient.SetBucketPolicy(context.Background(), bucketName, policy)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("Bucket %s is set to public\n", bucketName)
	}
}

// UploadToMinIO загружает файл в MinIO
func UploadToMinIO(ctx context.Context, filename string, src io.Reader, fileSize int64, mimeType string) error {
	_, err := MinioClient.PutObject(
		ctx,
		bucketName,
		filename,
		src,
		fileSize,
		minio.PutObjectOptions{ContentType: mimeType},
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "File upload error")
	}

	return nil
}

// DeleteFromMinIO удаляет файл из MinIO
func DeleteFromMinIO(ctx context.Context, filename string) error {
	// Проверяем, существует ли файл в хранилище
	_, err := MinioClient.StatObject(ctx, bucketName, filename, minio.StatObjectOptions{})
	if err != nil {
		// Если ошибка содержит "NoSuchKey", файл не найден
		var minioErr minio.ErrorResponse
		if errors.As(err, &minioErr) && minioErr.Code == "NoSuchKey" {
			return fiber.NewError(fiber.StatusNotFound, "File not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Error checking file existence")
	}
	if err = MinioClient.RemoveObject(ctx, bucketName, filename, minio.RemoveObjectOptions{}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete file from storage")
	}
	return nil
}

func GetUrl(filename string) string {
	return os.Getenv("MINIO_PUBLIC_ENDPOINT") + "/" + bucketName + "/" + filename
}
