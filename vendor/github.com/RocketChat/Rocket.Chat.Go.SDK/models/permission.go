package models

type Permission struct {
	ID        string   `json:"_id"`
	UpdatedAt string   `json:"_updatedAt.$date"`
	Roles     []string `json:"roles"`
}
