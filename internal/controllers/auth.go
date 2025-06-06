package controllers

import (
	"blogpoint-backend/internal/mail"
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"blogpoint-backend/internal/storage"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

const SecretKey = "secret"
const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Register регистрирует нового пользователя
// @Summary      Регистрация пользователя
// @Description  Регистрация нового пользователя
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        data  body      RegisterRequest true "Данные пользователя"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Router       /api/register [post]
func Register(c *fiber.Ctx) error {

	var data RegisterRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	if data.Login == "" || data.Email == "" || data.Password == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Incorrect data",
		})
	}

	if data.Language == "" {
		data.Language = "ru"
	} else if data.Language != "ru" && data.Language != "en" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Incorrect language value",
		})
	}

	var existingUser models.User
	if err := repository.DB.Where("login = ?", data.Login).First(&existingUser).Error; err == nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Login is already taken",
		})
	}

	// Проверка уникальности email
	if err := repository.DB.Where("email = ?", data.Email).First(&existingUser).Error; err == nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Email is already taken",
		})
	}

	password, _ := bcrypt.GenerateFromPassword([]byte(data.Password), 14)

	user := models.User{
		Login:    data.Login,
		Email:    data.Email,
		Password: password,
		Language: data.Language,
	}

	if err := repository.DB.Select("Login", "Email", "Password", "Language").Create(&user).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to create user",
		})
	}

	return c.JSON(MessageResponse{
		Message: "Successful registration",
	})
}

// Login авторизует пользователя
// @Summary      Авторизация
// @Description  Авторизация пользователя
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        data  body      LoginRequest true "Логин и пароль"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /api/login [post]
func Login(c *fiber.Ctx) error {
	var data LoginRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	var user models.User

	repository.DB.Where("login = ?", data.Login).First(&user)

	if user.Id == 0 {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "User not found",
		})
	}

	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(data.Password)); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Incorrect password",
		})
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": strconv.Itoa(int(user.Id)),
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	token, err := claims.SignedString([]byte(SecretKey))

	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Could not login",
		})
	}

	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(time.Hour),
		HTTPOnly: true,
	}

	c.Cookie(&cookie)

	return c.JSON(MessageResponse{
		Message: "Successful authorization",
	})
}

// Logout завершает сессию пользователя
// @Summary      Выход из аккаунта
// @Description  Удаляет JWT cookie и завершает сессию
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Router       /api/logout [post]
func Logout(c *fiber.Ctx) error {
	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	}

	c.Cookie(&cookie)

	return c.JSON(MessageResponse{
		Message: "Logged out successfully",
	})
}

// User возвращает текущего авторизованного пользователя
// @Summary      Получение данных пользователя
// @Description  Получение данных авторизованного пользователя
// @Tags         User
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  DataResponse[UserResponse]
// @Failure      401  {object}  ErrorResponse
// @Router       /api/user [get]
func User(c *fiber.Ctx) error {
	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	var user models.User

	repository.DB.Where("id = ?", claims["iss"]).First(&user)

	var logo *models.File
	if user.LogoId != nil {
		if err := repository.DB.First(&logo, *user.LogoId).Error; err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Error logo does not exist",
			})
		}

		if strings.Split(logo.MimeType, "/")[0] != "image" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Logo image file type is not allowed",
			})
		}
	}

	return c.JSON(DataResponse[UserResponse]{
		Data: ConvertUserToResponse(user, logo),
	})
}

// EditProfile обновляет профиль пользователя
// @Summary      Редактирование профиля
// @Description  Изменение профиля пользователя
// @Tags         User
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      EditProfileRequest true "Новые данные профиля"
// @Success      200   {object}  DataResponse[UserResponse]
// @Failure      401   {object}  ErrorResponse
// @Router       /api/editProfile [patch]
func EditProfile(c *fiber.Ctx) error {
	var data EditProfileRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	var user models.User
	repository.DB.Where("id = ?", claims["iss"]).First(&user)

	var logo *models.File
	if user.LogoId != nil {
		if err := repository.DB.First(&logo, *user.LogoId).Error; err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Error logo does not exist",
			})
		}

		if strings.Split(logo.MimeType, "/")[0] != "image" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Logo image file type is not allowed",
			})
		}
	}

	if data.Login != "" {
		user.Login = data.Login
	}
	if data.Email != "" {
		user.Email = data.Email
	}

	repository.DB.Save(&user)

	return c.JSON(DataResponse[UserResponse]{
		Data:    ConvertUserToResponse(user, logo),
		Message: "Profile edited successfully",
	})
}

// ChangePassword меняет пароль пользователя
// @Summary      Смена пароля
// @Description  Изменение пароля пользователя
// @Tags         User
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      ChangePasswordRequest true "Старый и новый пароль"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /api/changePassword [patch]
func ChangePassword(c *fiber.Ctx) error {
	var data ChangePasswordRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	var user models.User
	if err = repository.DB.Where("id = ?", claims["iss"]).First(&user).Error; err != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "User not found",
		})
	}

	if data.OldPassword == "" || data.NewPassword == "" || data.OldPassword == data.NewPassword {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Incorrect data",
		})
	}

	if err = bcrypt.CompareHashAndPassword(user.Password, []byte(data.OldPassword)); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Incorrect old password",
		})
	}
	password, err := bcrypt.GenerateFromPassword([]byte(data.NewPassword), 14)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to hash password",
		})
	}
	user.Password = password
	if err = repository.DB.Save(&user).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to update password",
		})
	}

	return c.JSON(MessageResponse{
		Message: "Password changed successfully",
	})
}

// LanguageUpdate меняет язык пользователя
// @Summary      Смена языка интерфейса
// @Description  Изменение языка интерфейса пользователя
// @Tags         User
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      LanguageUpdateRequest true "Новый язык интерфейса"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /api/languageUpdate [patch]
func LanguageUpdate(c *fiber.Ctx) error {
	var data LanguageUpdateRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	var user models.User
	if err = repository.DB.Where("id = ?", claims["iss"]).First(&user).Error; err != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "User not found",
		})
	}

	if user.Language == data.Language {
		return c.JSON(MessageResponse{
			Message: "Language is already set",
		})
	}

	if data.Language != "ru" && data.Language != "en" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Incorrect language value",
		})
	}

	user.Language = data.Language
	if err = repository.DB.Save(&user).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to update language",
		})
	}

	return c.JSON(MessageResponse{
		Message: "Language updated successfully",
	})
}

// UploadUserLogo загружает новое лого пользователя.
// @Summary      Загрузка лого пользователя
// @Description  Загружает изображение и устанавливает его как лого текущего авторизованного пользователя.
// @Tags         User
// @Security     ApiKeyAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file true "Файл изображения"
// @Success      200   {object}  DataResponse[FileResponse]
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/uploadUserLogo [post]
func UploadUserLogo(c *fiber.Ctx) error {
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

	var user models.User
	if err = repository.DB.First(&user, userId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Message: "User not found"})
	}

	var oldLogoId uint
	if user.LogoId != nil {
		oldLogoId = *user.LogoId
	}

	filename, mimeType, err := ProcessUpload(c, "image")

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

	if err = repository.DB.Model(&user).Update("logo_id", file.Id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Message: "Error saving logo id"})
	}

	if oldLogoId != 0 {
		var oldFile models.File
		if err = repository.DB.First(&oldFile, "id = ?", oldLogoId).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Message: "File not found",
			})
		}

		if err = storage.DeleteFromMinIO(c.Context(), oldFile.Filename); err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(ErrorResponse{
				Message: err.(*fiber.Error).Error(),
			})
		}

		if err = repository.DB.Delete(&oldFile).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Message: "Error deleting file from database",
			})
		}
	}

	url := storage.GetUrl(filename)

	fileResponse := FileResponse{
		Id:  file.Id,
		Url: url,
	}

	return c.JSON(DataResponse[FileResponse]{
		Data:    fileResponse,
		Message: "User logo uploaded successfully",
	})
}

// DeleteUserLogo удаляет лого пользователя
// @Summary      Удаление лого пользователя
// @Description  Удаляет текущее лого пользователя и очищает поле logo_id
// @Tags         User
// @Security     ApiKeyAuth
// @Accept       multipart/form-data
// @Produce      json
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /api/deleteUserLogo [delete]
func DeleteUserLogo(c *fiber.Ctx) error {
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

	var user models.User
	if err = repository.DB.First(&user, "id = ?", userId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Message: "User not found",
		})
	}

	if user.LogoId == nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "No logo to delete",
		})
	}

	var file models.File
	if err = repository.DB.First(&file, "id = ?", user.LogoId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Message: "File not found",
		})
	}

	if err = storage.DeleteFromMinIO(c.Context(), file.Filename); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(ErrorResponse{
			Message: err.(*fiber.Error).Error(),
		})
	}

	if err = repository.DB.Delete(&file).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Error deleting file from database",
		})
	}

	if err = repository.DB.Model(&user).Update("logo_id", nil).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Error clearing user logo",
		})
	}

	return c.JSON(MessageResponse{
		Message: "User logo deleted successfully",
	})
}

func generateCode() string {
	rand.Seed(time.Now().UnixNano())
	code := make([]byte, 6)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

// RequestEmailVerification отправляет код подтверждения на email
// @Summary      Запрос кода подтверждения email
// @Description  Отправление кода на почту для её подтверждения
// @Tags         User
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200 {object}  MessageResponse
// @Failure      401 {object}  ErrorResponse
// @Router       /api/requestEmailVerification [post]
func RequestEmailVerification(c *fiber.Ctx, emailSender mail.EmailSender) error {
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

	uintId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	// Удаляем старый код
	repository.DB.Delete(&models.VerificationCode{}, "user_id = ? AND type = ?", uintId, "email_verification")

	// Генерируем код
	code := generateCode()
	expiresAt := time.Now().Add(10 * time.Minute)

	verification := models.VerificationCode{
		UserId:    uint(uintId),
		Code:      code,
		Type:      "email_verification",
		ExpiresAt: expiresAt,
	}
	repository.DB.Create(&verification)

	// Отправляем email
	var user models.User
	repository.DB.First(&user, uintId)

	subject := "Blog point verification code"
	content := fmt.Sprintf(`
    <h1>Email Verification</h1>
    <p>Your verification code is: <strong>%s</strong></p>
    <p>Please enter this code on the website to verify your email.</p>
    <p>This code will expire in 10 minutes.</p>
    <p>Best regards, <br>Blog Point Team</p>`, code)

	to := []string{user.Email}

	err = emailSender.SendEmail(subject, content, to, nil, nil, nil)

	fmt.Println(emailSender)

	fmt.Println(err)

	if err != nil {
		return err
	}

	return c.JSON(MessageResponse{
		Message: "Verification code sent"})
}

// VerifyEmail проверяет код подтверждения email
// @Summary      Подтверждение email
// @Description  Подтверждение email полученным кодом
// @Tags         User
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      CodeRequest true "Код подтверждения"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /api/verifyEmail [post]
func VerifyEmail(c *fiber.Ctx) error {
	var data CodeRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	// Проверяем код
	var verification models.VerificationCode
	err = repository.DB.Where(
		"user_id = ? AND code = ? AND type = ? AND expires_at > ?",
		claims["iss"], data.Code, "email_verification", time.Now(),
	).First(&verification).Error

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid or expired code"})
	}

	// Подтверждаем email
	repository.DB.Model(&models.User{}).Where("id = ?", claims["iss"]).Update("is_verified", true)

	// Удаляем код
	repository.DB.Delete(&verification)

	return c.JSON(MessageResponse{
		Message: "Email verified successfully"})
}

// RequestPasswordReset отправляет ссылку для сброса пароля
// @Summary      Запрос на сброс пароля
// @Description  Отправление ссылки для сброса пароля на почту пользователя
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        data  body      EmailRequest true "Email пользователя"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /api/requestPasswordReset [post]
func RequestPasswordReset(c *fiber.Ctx, emailSender mail.EmailSender) error {
	var data EmailRequest
	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	email := data.Email
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Email is required"})
	}

	var user models.User
	if err := repository.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Email not found"})
	}

	repository.DB.Delete(&models.VerificationCode{}, "user_id = ? AND type = ?", user.Id, "password_reset")

	code := generateCode()
	expiresAt := time.Now().Add(10 * time.Minute)

	verification := models.VerificationCode{
		UserId:    user.Id,
		Code:      code,
		Type:      "password_reset",
		ExpiresAt: expiresAt,
	}
	repository.DB.Create(&verification)

	// Формируем ссылку
	resetLink := fmt.Sprintf("https://blogpoint.com/reset-password?token=%s", code)

	// Отправляем email
	subject := "Blog point password recovery"
	content := fmt.Sprintf(`
		<h1>Password recovery</h1>
		<p>Click the link below to reset your password:</p>
		<a href="%s">%s</a>
		<p>This link is valid for 10 minutes.</p>
		<p>Best regards, <br>Blog Point Team</p>`, resetLink, resetLink)

	to := []string{email}

	err := emailSender.SendEmail(subject, content, to, nil, nil, nil)

	if err != nil {
		return err
	}

	return c.JSON(MessageResponse{
		Message: "Password recovery link sent"})

}

// ResetPassword сбрасывает пароль
// @Summary      Сброс пароля
// @Description  Сброс пароля по коду, отправленному на почту
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        data  body      ResetPasswordRequest true "Код и новый пароль"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Router       /api/resetPassword [patch]
func ResetPassword(c *fiber.Ctx) error {
	var data ResetPasswordRequest
	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	code := data.Code
	newPassword := data.Password

	if code == "" || newPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request"})
	}
	if data.Code == "" || data.Password == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"message": "Incorrect data"})
	}

	var verification models.VerificationCode
	err := repository.DB.Where("code = ? AND type = ? AND expires_at > ?",
		data.Code, "password_reset", time.Now()).First(&verification).Error

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid or expired code"})
	}

	// Обновляем пароль
	var user models.User
	repository.DB.First(&user, verification.UserId)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), 14)
	user.Password = hashedPassword
	repository.DB.Save(&user)

	// Удаляем использованный токен
	repository.DB.Delete(&verification)

	return c.JSON(MessageResponse{
		Message: "Password changed successfully"})
}

// RequestDeletionVerification отправляет код для удаления аккаунта на почту
// @Summary      Отправить код подтверждения удаления аккаунта
// @Description  Отправляет код подтверждения на email для удаления аккаунта
// @Tags         User
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /api/requestDeletionVerification [post]
func RequestDeletionVerification(c *fiber.Ctx, emailSender mail.EmailSender) error {
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

	uintId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	// Удаляем старый код
	repository.DB.Delete(&models.VerificationCode{}, "user_id = ? AND type = ?", uintId, "account_deletion")

	// Генерируем код
	code := generateCode()
	expiresAt := time.Now().Add(10 * time.Minute)

	verification := models.VerificationCode{
		UserId:    uint(uintId),
		Code:      code,
		Type:      "account_deletion",
		ExpiresAt: expiresAt,
	}
	repository.DB.Create(&verification)

	// Отправляем email
	var user models.User
	repository.DB.First(&user, uintId)

	subject := "Blog point verification code"
	content := fmt.Sprintf(`
	<h1>Deletion confirmation</h1>
	<p>Your verification code is: <strong>%s</strong></p>
	<p>Please enter this code on the website to delete your account.</p>
	<p>This code will expire in 10 minutes.</p>
	<p>Best regards, <br>Blog Point Team</p>`, code)

	to := []string{user.Email}

	err = emailSender.SendEmail(subject, content, to, nil, nil, nil)

	if err != nil {
		return err
	}

	return c.JSON(MessageResponse{
		Message: "Deletion confirmation code sent"})
}

// DeleteUser удаляет аккаунт
// @Summary      Удаление аккаунта
// @Description  Удаляет аккаунт пользователя по коду подтверждения
// @Tags         User
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      CodeRequest true "Код подтверждения"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /api/deleteUser [delete]
func DeleteUser(c *fiber.Ctx) error {
	var data CodeRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	// Проверяем код
	var verification models.VerificationCode
	err = repository.DB.Where(
		"user_id = ? AND code = ? AND type = ? AND expires_at > ?",
		claims["iss"], data.Code, "account_deletion", time.Now(),
	).First(&verification).Error

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid or expired code"})
	}

	repository.DB.Delete(&models.User{}, "id = ?", claims["iss"])

	repository.DB.Delete(&verification)

	return c.JSON(MessageResponse{
		Message: "User deleted successfully",
	})
}

func ConvertUserToResponse(user models.User, file *models.File) UserResponse {
	var logo *FileResponse
	if file != nil {
		logo = &FileResponse{
			Id:  file.Id,
			Url: storage.GetUrl(file.Filename)}

	}

	return UserResponse{
		Id:         user.Id,
		Login:      user.Login,
		Email:      user.Email,
		Language:   user.Language,
		IsVerified: user.IsVerified,
		Logo:       logo,
	}
}
