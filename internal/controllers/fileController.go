package controllers

import (
	"blogpoint-backend/internal/storage"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

func GenerateUniqueFilename(filename string) string {
	ext := filepath.Ext(filename)    // Получаем расширение (.png, .jpg и т.д.)
	return uuid.New().String() + ext // Генерируем UUID и добавляем расширение
}

// UploadFileHandler обрабатывает загрузку файла
func UploadFileHandler(c fiber.Ctx) error {
	// Читаем файл из запроса
	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Ошибка при получении файла")
	}

	// Открываем файл
	src, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Ошибка открытия файла")
	}
	defer src.Close()

	// Определяем MIME-тип
	buffer := make([]byte, 512)
	if _, err := src.Read(buffer); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Ошибка при чтении файла")
	}
	// Возвращаемся в начало файла, так как Read сместил указатель
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Ошибка при сбросе указателя файла")
	}
	mimeType := http.DetectContentType(buffer)

	// Генерируем уникальное имя файла
	uniqueFilename := strings.Split(mimeType, "/")[0] + "/" + GenerateUniqueFilename(file.Filename)

	// Загружаем файл в MinIO
	url, err := storage.UploadFile(c.Context(), uniqueFilename, src, file.Size, mimeType)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Ошибка загрузки файла")
	}

	return c.JSON(fiber.Map{
		"filename": uniqueFilename,
		"url":      url,
	})
}

// DeleteFileHandler обрабатывает удаление файла
func DeleteFileHandler(c fiber.Ctx) error {
	filename := c.Query("filename")
	if filename == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Необходимо указать имя файла")
	}

	err := storage.DeleteFile(c.Context(), filename)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Файл удален"})
}
