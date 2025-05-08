package main

import (
	_ "blogpoint-backend/docs"
	"blogpoint-backend/internal/mail"
	"blogpoint-backend/internal/repository"
	"blogpoint-backend/internal/routes"
	"blogpoint-backend/internal/storage"
	"blogpoint-backend/utils"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

// @title BlogPoint API
// @version 1.0
// @description API для BlogPoint

// @host localhost:8000
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in cookie
// @name jwt
func main() {
	repository.Connect()
	storage.InitMinio()

	emailSender := mail.NewGmailSender("BlogPoint", "blogpointoff@gmail.com", "dacqkbzedkdnnxqu")

	app := fiber.New(fiber.Config{
		BodyLimit: 500 * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173",
		AllowCredentials: true,
	}))

	routes.Setup(app, emailSender)

	utils.StartCleanupTask()
	utils.StartStatisticsTask()

	app.Get("/swagger/*", swagger.HandlerDefault)

	err := app.Listen(":8000")

	if err != nil {
		fmt.Printf("fiber.Listen failed %s", err)
	}
}
