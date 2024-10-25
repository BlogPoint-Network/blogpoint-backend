package models

type User struct {
	Id       uint   `json:"id"`
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password []byte `json:"-"`
}
