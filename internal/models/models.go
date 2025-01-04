package models

type User struct {
	Id       uint   `json:"id"`
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password []byte `json:"-"`
}

type Channel struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerId     uint   `json:"ownerId"`
	SubsCount   uint   `json:"subsCount"`
}

type Post struct {
	Id            uint   `json:"id"`
	ChannelId     uint   `json:"channelId"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	LikesCount    uint   `json:"likesCount"`
	DislikesCount uint   `json:"dislikesCount"`
	ViewsCount    uint   `json:"viewsCount"`
}

type Subscription struct {
	UserId    uint `json:"userId"`
	ChannelId uint `json:"channelId"`
}
