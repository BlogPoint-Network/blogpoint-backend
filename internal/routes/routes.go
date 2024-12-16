package routes

import (
	"blogpoint-backend/internal/controllers"
	"github.com/gofiber/fiber/v3"
)

func Setup(app *fiber.App) {

	app.Post("/api/register", controllers.Register)
	app.Post("/api/login", controllers.Login)
	app.Post("/api/user", controllers.User)
	app.Post("/api/editprofile", controllers.EditProfile)
	app.Post("/api/deleteprofile", controllers.DeleteUser)

	app.Post("/api/createchannel", controllers.CreateChannel)
	app.Post("/api/editchannel", controllers.EditChannel)
	app.Post("/api/deletechannel", controllers.DeleteChannel)
	app.Post("/api/getusersubscriptions", controllers.GetUserSubscriptions)
	app.Post("/api/getuserchannels", controllers.GetUserChannels)
	app.Post("/api/getChannel", controllers.GetChannel)

	app.Post("/api/createblog", controllers.CreateBlog)
	app.Post("/api/editblog", controllers.EditBlog)
	app.Post("/api/deleteblog", controllers.DeleteBlog)
}
