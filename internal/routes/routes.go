package routes

import (
	"blogpoint-backend/internal/controllers"
	"blogpoint-backend/internal/mail"
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, emailSender mail.EmailSender) {

	app.Post("/api/register", controllers.Register)
	app.Post("/api/login", controllers.Login)
	app.Post("/api/logout", controllers.Logout)
	app.Get("/api/user", controllers.User)
	app.Patch("/api/editProfile", controllers.EditProfile)
	app.Patch("/api/changePassword", controllers.ChangePassword)
	app.Patch("/api/languageUpdate", controllers.LanguageUpdate)

	app.Post("/api/requestEmailVerification", func(c *fiber.Ctx) error {
		return controllers.RequestEmailVerification(c, emailSender)
	})
	app.Post("/api/verifyEmail", controllers.VerifyEmail)
	app.Post("/api/requestPasswordReset", func(c *fiber.Ctx) error {
		return controllers.RequestPasswordReset(c, emailSender)
	})
	app.Patch("/api/resetPassword", controllers.ResetPassword)
	app.Post("/api/requestDeletionVerification", func(c *fiber.Ctx) error {
		return controllers.RequestDeletionVerification(c, emailSender)
	})
	app.Delete("/api/deleteUser", controllers.DeleteUser)

	app.Post("/api/createChannel", controllers.CreateChannel)
	app.Patch("/api/editChannel", controllers.EditChannel)
	app.Delete("/api/deleteChannel/:id", controllers.DeleteChannel)
	app.Get("/api/getUserSubscriptions", controllers.GetUserSubscriptions)
	app.Get("/api/getUserChannels", controllers.GetUserChannels)
	app.Get("/api/getChannel/:id", controllers.GetChannel)
	app.Get("/api/getPopularChannels", controllers.GetPopularChannels)
	app.Post("/api/subscribeChannel/:id", controllers.SubscribeChannel)
	app.Delete("/api/unsubscribeChannel/:id", controllers.UnsubscribeChannel)
	app.Get("/api/getChannelStatistics/:id", controllers.GetChannelStatistics)

	app.Get("/api/getAllCategories", controllers.GetAllCategories)
	app.Get("/api/getAllTags", controllers.GetAllTags)

	app.Post("/api/createPost", controllers.CreatePost)
	app.Patch("/api/editPost", controllers.EditPost)
	app.Delete("/api/deletePost/:id", controllers.DeletePost)
	app.Get("/api/getPost/:id", controllers.GetPost)
	app.Get("/api/getPosts/:channelId", controllers.GetPosts)
	app.Post("/api/setReaction", controllers.SetReaction)
	app.Post("/api/createComment", controllers.CreateComment)
	app.Get("/api/getChildComments/:id", controllers.GetChildComments)
	app.Delete("/api/deleteComment/:id", controllers.DeleteComment)

	app.Post("/api/uploadFile", controllers.UploadFileHandler)
	app.Delete("/api/deleteFile", controllers.DeleteFileHandler)
}
