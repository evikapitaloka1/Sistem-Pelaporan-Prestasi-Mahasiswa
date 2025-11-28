package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	FullName     string    `json:"full_name"`
	RoleID       uuid.UUID `json:"role_id"`
	RoleName     string    `json:"role_name"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	IsActive    bool      `json:"is_active"`
}

type CreateUserRequest struct {
	Username string    `json:"username" validate:"required"`
	Email    string    `json:"email" validate:"required,email"`
	Password string    `json:"password" validate:"required,min=6"`
	FullName string    `json:"full_name" validate:"required"`
	RoleID   uuid.UUID `json:"role_id" validate:"required"`
}

type UpdateUserRequest struct {
	Username string    `json:"username"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	RoleID   uuid.UUID `json:"role_id"`
	IsActive bool      `json:"is_active"`
}

type UpdateUserRoleRequest struct {
	RoleID uuid.UUID `json:"role_id" validate:"required"`
}

type EmptyRequest struct{}
