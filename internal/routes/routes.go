package routes

import (
	"blogpoint-backend/internal/controllers"
	"github.com/gofiber/fiber/v3"
)

func Setup(app *fiber.App) {

	app.Post("/api/uploadfile", controllers.UploadFileHandler)
	app.Delete("/api/deletefile", controllers.DeleteFileHandler)

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
	app.Post("/api/subscribechannel", controllers.SubscribeChannel)
	app.Post("/api/unsubscribechannel", controllers.UnsubscribeChannel)

	app.Post("/api/createpost", controllers.CreatePost)
	app.Post("/api/editpost", controllers.EditPost)
	app.Post("/api/deletepost", controllers.DeletePost)
	app.Post("/api/getpost", controllers.GetPost)
	app.Post("/api/getposts", controllers.GetPosts)

}
