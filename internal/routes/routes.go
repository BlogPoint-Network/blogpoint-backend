package routes

import (
	"blogpoint-backend/internal/controllers"
	"blogpoint-backend/internal/mail"
	"github.com/gofiber/fiber/v3"
)

func Setup(app *fiber.App, emailSender mail.EmailSender) {

	app.Post("/api/uploadfile", controllers.UploadFileHandler)
	app.Delete("/api/deletefile", controllers.DeleteFileHandler)

	app.Post("/api/register", controllers.Register)
	app.Post("/api/requestemailverification", func(c fiber.Ctx) error {
		return controllers.RequestEmailVerification(c, emailSender)
	})
	app.Post("/api/verifyemail", controllers.VerifyEmail)
	app.Post("/api/login", controllers.Login)
	app.Get("/api/user", controllers.User)
	app.Patch("/api/editprofile", controllers.EditProfile)
	app.Post("/api/requestdeletionverification", func(c fiber.Ctx) error {
		return controllers.RequestDeletionVerification(c, emailSender)
	})
	app.Delete("/api/deleteprofile", controllers.DeleteUser)
	app.Post("/api/requestpasswordreset", func(c fiber.Ctx) error {
		return controllers.RequestPasswordReset(c, emailSender)
	})
	app.Patch("/api/resetpassword", controllers.ResetPassword)

	app.Post("/api/createchannel", controllers.CreateChannel)
	app.Patch("/api/editchannel", controllers.EditChannel)
	app.Delete("/api/deletechannel", controllers.DeleteChannel)
	app.Get("/api/getusersubscriptions", controllers.GetUserSubscriptions)
	app.Get("/api/getuserchannels", controllers.GetUserChannels)
	app.Get("/api/getChannel", controllers.GetChannel)
	app.Get("/api/getPopularChannels", controllers.GetPopularChannels)
	app.Post("/api/subscribechannel", controllers.SubscribeChannel)
	app.Delete("/api/unsubscribechannel", controllers.UnsubscribeChannel)

	app.Post("/api/createpost", controllers.CreatePost)
	app.Patch("/api/editpost", controllers.EditPost)
	app.Delete("/api/deletepost", controllers.DeletePost)
	app.Get("/api/getpost", controllers.GetPost)
	app.Get("/api/getposts", controllers.GetPosts)
	app.Post("/api/setreaction", controllers.SetReaction)
}
