package models

import "time"

type User struct {
	Id         uint   `json:"id"`
	Login      string `json:"login"`
	Email      string `json:"email"`
	Password   []byte `json:"-"`
	IsVerified bool   `json:"isVerified"`
}

type VerificationCode struct {
	Id        uint      `json:"id"`
	UserId    uint      `json:"userId"`
	Code      string    `json:"code"`
	Type      string    `json:"type"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Channel struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerId     uint   `json:"ownerId"`
	SubsCount   uint   `json:"subsCount"`
	LogoId      *uint  `json:"logoId"`
	BannerId    *uint  `json:"bannerId"`
}

type Post struct {
	Id            uint   `json:"id"`
	ChannelId     uint   `json:"channelId"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	LikesCount    uint   `json:"likesCount"`
	DislikesCount uint   `json:"dislikesCount"`
	ViewsCount    uint   `json:"viewsCount"`
	Tags          []Tag  `gorm:"many2many:post_tags;" json:"tags"`
}

type Category struct {
	Id    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Tags  []Tag  `json:"tags"`
}

type Tag struct {
	Id         uint   `json:"id"`
	CategoryId uint   `json:"categoryId"`
	Name       string `json:"name"`
	Color      string `json:"color"`
}

type PostTag struct {
	PostId uint `json:"postId"`
	TagId  uint `json:"tagId"`
}

type PostReaction struct {
	Id       uint `json:"id"`
	PostId   uint `json:"postId"`
	UserId   uint `json:"userId"`
	Reaction bool `json:"reaction"`
}

type Subscription struct {
	UserId    uint `json:"userId"`
	ChannelId uint `json:"channelId"`
}

type File struct {
	Id       uint   `json:"id"`
	UserId   uint   `json:"channelId"`
	Filename string `json:"title"`
	MimeType string `json:"MimeType"`
}
