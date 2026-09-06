package model

import "time"

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserDTO struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	FullName string   `json:"fullName"`
	TenantID string   `json:"tenantId"`
	Role     string   `json:"role"`
	Modules  []string `json:"modules"`
}

type LoginResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

type RequestMeta struct {
	IP        string
	UserAgent string
}

type DirectoryEntry struct {
	TenantID string
	UserID   string
}

type TenantStatus struct {
	IsActive bool
	SubEnd   *time.Time
	Modules  []string
}

type DBUser struct {
	Email        string
	PasswordHash string
	FullName     string
	Role         string
}
