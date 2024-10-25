package main

import (
	"blogpoint-backend/internal/repository"
	"blogpoint-backend/internal/routes"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	repository.Connect()

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowCredentials: true,
	}))

	routes.Setup(app)

	app.Listen(":8000")
}
