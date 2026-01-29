package models

type UserCredentials struct {
	ID    string `json:"id"`
	Token string `json:"token"`

	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"pass"`
}
