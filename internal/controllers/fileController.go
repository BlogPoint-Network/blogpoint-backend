package controllers

import (
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"blogpoint-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func GenerateUniqueFilename(filename string) string {
	ext := filepath.Ext(filename)    // Получаем расширение (.png, .jpg и т.д.)
	return uuid.New().String() + ext // Генерируем UUID и добавляем расширение
}

// UploadFile обрабатывает загрузку файла
// @Summary      Загрузка файла
// @Description  Загружает файл и возвращает его URL и уникальное имя
// @Tags         File
// @Security     ApiKeyAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file true "Файл для загрузки"
// @Success      200   {object}  FileResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/uploadFile [post]
func UploadFile(c *fiber.Ctx) error {
	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid token",
		})
	}

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid user Id",
		})
	}

	filename, mimeType, err := ProcessUpload(c, "")

	if err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(ErrorResponse{
			Message: err.(*fiber.Error).Error(),
		})
	}

	file := models.File{
		OwnerId:  uint(userId),
		Filename: filename,
		MimeType: mimeType,
	}

	if err = repository.DB.Create(&file).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Message: "Error saving file to DB"})
	}

	url := storage.GetUrl(filename)

	fileResponse := FileResponse{
		Id:  file.Id,
		Url: url,
	}

	return c.JSON(DataResponse[FileResponse]{
		Data:    fileResponse,
		Message: "Файл загружен",
	})
}

// ProcessUpload обрабатывает загрузку файла в MinIo
func ProcessUpload(c *fiber.Ctx, allowedType string) (string, string, error) {
	// Читаем файл из запроса
	file, err := c.FormFile("file")
	if err != nil {
		return "", "", fiber.NewError(fiber.StatusBadRequest, "Error receiving file")
	}

	// Открываем файл
	src, err := file.Open()
	if err != nil {
		return "", "", fiber.NewError(fiber.StatusInternalServerError, "Error opening file")
	}
	defer src.Close()

	// Определяем MIME-тип
	buffer := make([]byte, 512)
	if _, err := src.Read(buffer); err != nil {
		return "", "", fiber.NewError(fiber.StatusInternalServerError, "Error reading file")
	}
	// Возвращаемся в начало файла, так как Read сместил указатель
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", "", fiber.NewError(fiber.StatusInternalServerError, "Error resetting file pointer")
	}
	mimeType := http.DetectContentType(buffer)

	if allowedType != "" && allowedType != strings.Split(mimeType, "/")[0] {
		return "", "", fiber.NewError(fiber.StatusBadRequest, "File type is not allowed")
	}

	// Генерируем уникальное имя файла
	uniqueFilename := strings.Split(mimeType, "/")[0] + "/" + GenerateUniqueFilename(file.Filename)

	// Загружаем файл в MinIO
	if err = storage.UploadToMinIO(c.Context(), uniqueFilename, src, file.Size, mimeType); err != nil {
		return "", "", err
	}

	return uniqueFilename, mimeType, nil
}

// DeleteFile обрабатывает удаление файла
// @Summary      Удаление файла
// @Description  Удаляет файл по id
// @Tags         File
// @Security     ApiKeyAuth
// @Produce      json
// @Param        id   path      int true "Id файла для удаления"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/deleteFile/{id} [delete]
func DeleteFile(c *fiber.Ctx) error {
	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid token",
		})
	}

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	fileId, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || fileId == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid file id",
		})
	}

	var file models.File
	if result := repository.DB.First(&file, fileId); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "File not found",
		})
	}

	if file.OwnerId != uint(userId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(ErrorResponse{
			Message: "You are not the owner of this file",
		})
	}

	if err = storage.DeleteFromMinIO(c.Context(), file.Filename); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(ErrorResponse{
			Message: err.(*fiber.Error).Error(),
		})
	}

	if err := repository.DB.Delete(&file).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to delete file from DB",
		})
	}

	return c.JSON(MessageResponse{
		Message: "File deleted successfully",
	})
}
