package models

type User struct {
	ID           string `json:"_id"`
	Name         string `json:"name"`
	UserName     string `json:"username"`
	Status       string `json:"status"`
	Token        string `json:"token"`
	TokenExpires int64  `json:"tokenExpires"`
}

type CreateUserRequest struct {
	Name         string            `json:"name"`
	Email        string            `json:"email"`
	Password     string            `json:"password"`
	Username     string            `json:"username"`
	Roles        []string          `json:"roles,omitempty"`
	CustomFields map[string]string `json:"customFields,omitempty"`
}

type UpdateUserRequest struct {
	UserID string `json:"userId"`
	Data   struct {
		Name         string            `json:"name"`
		Email        string            `json:"email"`
		Password     string            `json:"password"`
		Username     string            `json:"username"`
		CustomFields map[string]string `json:"customFields,omitempty"`
	} `json:"data"`
}
