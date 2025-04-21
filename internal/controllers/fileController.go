package controllers

import (
	"blogpoint-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
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
// @Summary Загрузка файла
// @Description Загружает файл и возвращает его URL и уникальное имя
// @Tags File
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Файл для загрузки"
// @Success 200 {object} FileResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/uploadFile [post]
func UploadFileHandler(c *fiber.Ctx) error {
	// Читаем файл из запроса
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Ошибка при получении файла",
		})
	}

	// Открываем файл
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Ошибка открытия файла",
		})
	}
	defer src.Close()

	// Определяем MIME-тип
	buffer := make([]byte, 512)
	if _, err := src.Read(buffer); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Ошибка при чтении файла",
		})
	}
	// Возвращаемся в начало файла, так как Read сместил указатель
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Ошибка при сбросе указателя файла",
		})
	}
	mimeType := http.DetectContentType(buffer)

	// Генерируем уникальное имя файла
	uniqueFilename := strings.Split(mimeType, "/")[0] + "/" + GenerateUniqueFilename(file.Filename)

	// Загружаем файл в MinIO
	url, err := storage.UploadFile(c.Context(), uniqueFilename, src, file.Size, mimeType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Ошибка загрузки файла",
		})
	}

	return c.JSON(FileResponse{
		Filename: uniqueFilename,
		Url:      url,
	})
}

// DeleteFileHandler обрабатывает удаление файла
// @Summary Удаление файла
// @Description Удаляет файл по имени
// @Tags File
// @Security ApiKeyAuth
// @Produce json
// @Param filename query string true "Имя файла для удаления"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/deleteFile [delete]
func DeleteFileHandler(c *fiber.Ctx) error {
	filename := c.Query("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Необходимо указать имя файла",
		})
	}

	err := storage.DeleteFile(c.Context(), filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Ошибка удаления файла",
		})
	}

	return c.JSON(MessageResponse{
		Message: "Файл удален"})
}
