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
	app.Post("/api/user", controllers.User)
	app.Post("/api/editprofile", controllers.EditProfile)
	app.Post("/api/requestdeletionverification", func(c fiber.Ctx) error {
		return controllers.RequestDeletionVerification(c, emailSender)
	})
	app.Post("/api/deleteprofile", controllers.DeleteUser)
	app.Post("/api/requestpasswordreset", func(c fiber.Ctx) error {
		return controllers.RequestPasswordReset(c, emailSender)
	})
	app.Post("/api/resetpassword", controllers.ResetPassword)

	app.Post("/api/createchannel", controllers.CreateChannel)
	app.Post("/api/editchannel", controllers.EditChannel)
	app.Post("/api/deletechannel", controllers.DeleteChannel)
	app.Post("/api/getusersubscriptions", controllers.GetUserSubscriptions)
	app.Post("/api/getuserchannels", controllers.GetUserChannels)
	app.Post("/api/getChannel", controllers.GetChannel)
	app.Post("/api/getPopularChannels", controllers.GetPopularChannels)
	app.Post("/api/subscribechannel", controllers.SubscribeChannel)
	app.Post("/api/unsubscribechannel", controllers.UnsubscribeChannel)

	app.Post("/api/createpost", controllers.CreatePost)
	app.Post("/api/editpost", controllers.EditPost)
	app.Post("/api/deletepost", controllers.DeletePost)
	app.Post("/api/getpost", controllers.GetPost)
	app.Post("/api/getposts", controllers.GetPosts)
	app.Post("/api/setreaction", controllers.SetReaction)
}
