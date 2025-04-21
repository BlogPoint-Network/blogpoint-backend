package controllers

type DataResponse[T any] struct {
	Data    T      `json:"data"`
	Message string `json:"message,omitempty"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type FileResponse struct {
	Filename string `json:"message"`
	Url      string `json:"url"`
}

type ErrorResponse struct {
	Message string `json:"message" example:"Example"`
}

type RegisterRequest struct {
	Login    string `json:"login" example:"johndoe"`
	Password string `json:"password" example:"secret123"`
	Email    string `json:"email" example:"user@example.com"`
}

type LoginRequest struct {
	Login    string `json:"login" example:"johndoe"`
	Password string `json:"password" example:"secret123"`
}

type CodeRequest struct {
	Code string `json:"code" example:"H4RF1G"`
}

type EditProfileRequest struct {
	Login string `json:"login" example:"johndoe"`
	Email string `json:"email" example:"user@example.com"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" example:"oldSecret123"`
	NewPassword string `json:"newPassword" example:"newSecret123"`
}

type EmailRequest struct {
	Email string `json:"email" example:"user@example.com"`
}

type ResetPasswordRequest struct {
	Code     string `json:"code" example:"H4RF1G"`
	Password string `json:"password" example:"secret123"`
}

type CreateChannelRequest struct {
	Name        string `json:"name" example:"BlogPoint News"`
	Description string `json:"description" example:"More blogs here"`
	CategoryId  *uint  `json:"categoryId" example:"12"`
}

type EditChannelRequest struct {
	ChannelId   int    `json:"channelId" example:"1"`
	Name        string `json:"name" example:"BlogPoint News"`
	Description string `json:"description" example:"More blogs here"`
	CategoryId  *uint  `json:"categoryId" example:"12"`
}

type CreatePostRequest struct {
	ChannelId int    `json:"channelId" example:"1"`
	Title     string `json:"title" example:"Today's news"`
	Content   string `json:"content" example:"Something here"`
	Tags      string `json:"tags" example:"Новости, Журналистика, Статьи"`
}

type EditPostRequest struct {
	PostId  int    `json:"postId" example:"1"`
	Title   string `json:"title" example:"Today's news"`
	Content string `json:"content" example:"Something here"`
	Tags    string `json:"tags" example:"Новости, Журналистика, Статьи"`
}

type SetReactionRequest struct {
	PostId   int    `json:"postId" example:"1"`
	Reaction string `json:"reaction" example:"like"`
}
