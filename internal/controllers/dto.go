package controllers

import "blogpoint-backend/internal/models"

type DataResponse[T any] struct {
	Data    T      `json:"data"`
	Message string `json:"message,omitempty"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type FileResponse struct {
	Id  uint   `json:"id"`
	Url string `json:"url"`
}

type ErrorResponse struct {
	Message string `json:"message" example:"Example"`
}

type PostResponse struct {
	models.Post
	Comments []CommentResponse `json:"comments"`
}

type CommentResponse struct {
	Id           uint   `json:"id"`
	PostId       uint   `json:"postId"`
	ParentId     *uint  `json:"parentId,omitempty"`
	Content      string `json:"content"`
	IsDeleted    bool   `json:"isDeleted"`
	RepliesCount int    `json:"repliesCount"`
	User         struct {
		Id       uint   `json:"id"`
		Username string `json:"username"`
		//AvatarURL string `json:"avatarUrl"`
	} `json:"user"`
}

type StatisticsResponse struct {
	Current ChannelStatistics `json:"current"`
	Delta   ChannelStatistics `json:"delta"`
}

type ChannelStatistics struct {
	Views    int `json:"views" example:"12"`
	Likes    int `json:"likes" example:"5"`
	Dislikes int `json:"dislikes" example:"3"`
	Posts    int `json:"posts" example:"1"`
	Comments int `json:"comments" example:"2"`
}

type CategoryResponse struct {
	Id    uint   `json:"id" example:"1"`
	Name  string `json:"name" example:"Личный блог"`
	Color string `json:"color" example:"#FF9800"`
}

type TagResponse struct {
	Id         uint   `json:"id" example:"2"`
	CategoryId uint   `json:"categoryId" example:"11"`
	Name       string `json:"name" example:"Мотивация"`
	Color      string `json:"color" example:"#FF9800"`
}

type RegisterRequest struct {
	Login    string `json:"login" example:"johndoe"`
	Password string `json:"password" example:"secret123"`
	Email    string `json:"email" example:"user@example.com"`
	Language string `json:"language" example:"ru"`
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
	ChannelId   uint   `json:"channelId" example:"1"`
	Name        string `json:"name" example:"BlogPoint News"`
	Description string `json:"description" example:"More blogs here"`
	CategoryId  *uint  `json:"categoryId" example:"12"`
}

type LanguageUpdateRequest struct {
	Language string `json:"language" example:"ru"`
}

type CreatePostRequest struct {
	ChannelId  uint   `json:"channelId" example:"1"`
	Title      string `json:"title" example:"Today's news"`
	Content    string `json:"content" example:"Something here"`
	Tags       []uint `json:"tags"`
	PostImages []uint `json:"postImages"`
	PostFiles  []uint `json:"postFiles"`
}

type EditPostRequest struct {
	PostId  uint   `json:"postId" example:"1"`
	Title   string `json:"title" example:"Today's news"`
	Content string `json:"content" example:"Something here"`
	Tags    []uint `json:"tags"`
}

type SetReactionRequest struct {
	PostId   uint   `json:"postId" example:"1"`
	Reaction string `json:"reaction" example:"like"`
}

type CreateCommentRequest struct {
	PostId   uint   `json:"postId" example:"1"`
	Content  string `json:"content" example:"1"`
	ParentId *uint  `json:"parentId" example:"1"`
}
