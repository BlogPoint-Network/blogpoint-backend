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
		log.Fatalf("Ошибка инициализации MinIO: %v", err)
	}

	// Проверяем существование бакета
	exists, err := MinioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		log.Fatalf("Ошибка проверки бакета: %v", err)
	}

	// Создаем бакет, если его нет
	if !exists {
		err = MinioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Ошибка создания бакета: %v", err)
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
		fmt.Printf("Бакет %s установлен публичным\n", bucketName)
	}
}

// UploadFile загружает файл в MinIO
func UploadFile(ctx context.Context, filename string, src io.Reader, fileSize int64, mimeType string) (string, error) {
	_, err := MinioClient.PutObject(
		ctx,
		bucketName,
		filename,
		src,
		fileSize,
		minio.PutObjectOptions{ContentType: mimeType},
	)
	if err != nil {
		return "", fmt.Errorf("ошибка загрузки файла: %w", err)
	}

	// Генерируем ссылку на файл
	fileURL := os.Getenv("MINIO_PUBLIC_ENDPOINT") + "/" + bucketName + "/" + filename

	fmt.Println(fileURL)

	return fileURL, nil
}

// DeleteFile удаляет файл из MinIO
func DeleteFile(ctx context.Context, filename string) error {
	// Проверяем, существует ли файл в хранилище
	_, err := MinioClient.StatObject(ctx, bucketName, filename, minio.StatObjectOptions{})
	if err != nil {
		// Если ошибка содержит "NoSuchKey", файл не найден
		var minioErr minio.ErrorResponse
		if errors.As(err, &minioErr) && minioErr.Code == "NoSuchKey" {
			return fiber.NewError(fiber.StatusNotFound, "Файл не найден")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Ошибка при проверке существования файла")
	}
	return MinioClient.RemoveObject(ctx, bucketName, filename, minio.RemoveObjectOptions{})
}
