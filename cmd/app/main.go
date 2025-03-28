package main

import (
	"blogpoint-backend/internal/mail"
	"blogpoint-backend/internal/repository"
	"blogpoint-backend/internal/routes"
	"blogpoint-backend/internal/storage"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	repository.Connect()
	storage.InitMinio()

	emailSender := mail.NewGmailSender("BlogPoint", "blogpointoff@gmail.com", "dacqkbzedkdnnxqu")

	app := fiber.New(fiber.Config{
		BodyLimit: 500 * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowCredentials: true,
	}))

	routes.Setup(app, emailSender)

	app.Listen(":8000")
}
