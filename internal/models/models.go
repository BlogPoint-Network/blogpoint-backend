package models

import "time"

type User struct {
	Id         uint   `json:"id"`
	Login      string `json:"login"`
	Email      string `json:"email"`
	Password   []byte `json:"-"`
	Language   string `json:"language"`
	IsVerified bool   `json:"isVerified"`
	LogoId     *uint  `json:"logoId"`
}

type VerificationCode struct {
	Id        uint      `json:"id"`
	UserId    uint      `json:"userId"`
	Code      string    `json:"code"`
	Type      string    `json:"type"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Channel struct {
	Id          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CategoryId  *uint     `json:"-"`
	Category    *Category `json:"category"`
	OwnerId     uint      `json:"ownerId"`
	SubsCount   uint      `json:"subsCount"`
	LogoId      *uint     `json:"logoId"`
}

type Post struct {
	Id             uint      `json:"id"`
	ChannelId      uint      `json:"channelId"`
	PreviewImageId *uint     `json:"previewImage"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	LikesCount     uint      `json:"likesCount"`
	DislikesCount  uint      `json:"dislikesCount"`
	ViewsCount     uint      `json:"viewsCount"`
	PostImages     []File    `gorm:"many2many:post_images;" json:"postImages"`
	PostFiles      []File    `gorm:"many2many:post_files;" json:"postFiles"`
	Tags           []Tag     `gorm:"many2many:post_tags;" json:"tags"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Category struct {
	Id    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Tag struct {
	Id         uint   `json:"id"`
	CategoryId uint   `json:"categoryId"`
	Name       string `json:"name"`
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

type Comment struct {
	Id        uint      `json:"id"`
	ParentId  *uint     `json:"parentId,omitempty"`
	PostId    uint      `json:"postId"`
	UserId    uint      `json:"userId"`
	Content   string    `json:"content"`
	IsDeleted bool      `json:"isDeleted"`
	CreatedAt time.Time `json:"createdAt"`
}

type Subscription struct {
	UserId    uint `json:"userId"`
	ChannelId uint `json:"channelId"`
}

type File struct {
	Id       uint   `json:"id"`
	OwnerId  uint   `json:"ownerId"`
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
}

type ChannelStatistics struct {
	Id        uint `json:"id"`
	ChannelId uint `json:"ChannelId"`
	Views     int  `json:"Views"`
	Likes     int  `json:"Likes"`
	Dislikes  int  `json:"Dislikes"`
	Posts     int  `json:"Posts"`
	Comments  int  `json:"Comments"`
	Date      time.Time
}
