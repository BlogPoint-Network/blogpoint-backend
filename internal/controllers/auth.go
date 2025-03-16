package controllers

import (
	"blogpoint-backend/internal/mail"
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"strconv"
	"time"
)

const SecretKey = "secret"

func Register(c fiber.Ctx) error {

	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	if data["login"] == "" || data["email"] == "" || data["password"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Incorrect data",
		})
	}

	password, _ := bcrypt.GenerateFromPassword([]byte(data["password"]), 14)

	user := models.User{
		Login:    data["login"],
		Email:    data["email"],
		Password: password,
	}

	repository.DB.Select("Login", "Email", "Password").Create(&user)

	return c.JSON(user)
}

func generateCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000)) // Генерирует 6-значный код
}

func RequestEmailVerification(c fiber.Ctx, emailSender mail.EmailSender) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid token",
		})
	}

	uintId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid issuer id",
		})
	}

	// Удаляем старый код
	repository.DB.Delete(&models.VerificationCode{}, "user_id = ? AND purpose = ?", uintId, "email_verification")

	// Генерируем код
	code := generateCode()
	expiresAt := time.Now().Add(10 * time.Minute)

	verification := models.VerificationCode{
		UserId:    uint(uintId),
		Code:      code,
		Purpose:   "email_verification",
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

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Verification code sent"})
}

func VerifyEmail(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	// Проверяем код
	var verification models.VerificationCode
	err = repository.DB.Where(
		"user_id = ? AND code = ? AND purpose = ? AND expires_at > ?",
		claims["iss"], data["code"], "email_verification", time.Now(),
	).First(&verification).Error

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid or expired code"})
	}

	// Подтверждаем email
	repository.DB.Model(&models.User{}).Where("id = ?", claims["iss"]).Update("is_verified", true)

	// Удаляем код
	repository.DB.Delete(&verification)

	return c.JSON(fiber.Map{"message": "Email verified successfully"})
}

func Login(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	var user models.User

	repository.DB.Where("login = ?", data["login"]).First(&user)

	if user.Id == 0 {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "User not found",
		})
	}

	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(data["password"])); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Incorrect password",
		})
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": strconv.Itoa(int(user.Id)),
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	token, err := claims.SignedString([]byte(SecretKey))

	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Could not login",
		})
	}

	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(time.Hour),
		HTTPOnly: true,
	}

	c.Cookie(&cookie)

	return c.JSON(fiber.Map{
		"token": token,
		"user":  user,
	})
}

func User(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	var user models.User

	repository.DB.Where("id = ?", claims["iss"]).First(&user)

	return c.JSON(user)
}

func EditProfile(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	var user models.User
	repository.DB.Where("id = ?", claims["iss"]).First(&user)

	if login, ok := data["login"]; ok {
		user.Login = login
	}
	if email, ok := data["email"]; ok {
		user.Email = email
	}

	repository.DB.Save(&user)

	return c.JSON(user)
}

func RequestDeletionVerification(c fiber.Ctx, emailSender mail.EmailSender) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid token",
		})
	}

	uintId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid issuer id",
		})
	}

	// Удаляем старый код
	repository.DB.Delete(&models.VerificationCode{}, "user_id = ? AND purpose = ?", uintId, "account_deletion")

	// Генерируем код
	code := generateCode()
	expiresAt := time.Now().Add(10 * time.Minute)

	verification := models.VerificationCode{
		UserId:    uint(uintId),
		Code:      code,
		Purpose:   "account_deletion",
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

	return c.JSON(fiber.Map{"message": "Deletion confirmation code sent"})
}

func DeleteUser(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	// Проверяем код
	var verification models.VerificationCode
	err = repository.DB.Where(
		"user_id = ? AND code = ? AND purpose = ? AND expires_at > ?",
		claims["iss"], data["code"], "account_deletion", time.Now(),
	).First(&verification).Error

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid or expired code"})
	}

	repository.DB.Delete(&models.User{}, "id = ?", claims["iss"])

	repository.DB.Delete(&verification)

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}
