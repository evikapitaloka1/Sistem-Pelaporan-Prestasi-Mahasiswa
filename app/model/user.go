package model

import (
	"time"

	"github.com/google/uuid"
)
// 2. Main User Table


type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Tidak di-return di JSON response
	FullName     string    `json:"full_name"`
	RoleID       uuid.UUID `json:"role_id"`
	Role         Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type LoginResponse struct {
    Token        string      `json:"token"`
    RefreshToken string      `json:"refreshToken"`
    User         UserResponse `json:"user"` // Gunakan struct khusus biar rapi
}

// Struktur User khusus untuk Response Login (Sesuai Appendix)
type UserResponse struct {
    ID          string   `json:"id"`
    Username    string   `json:"username"`
    FullName    string   `json:"fullName"` // [PERBAIKAN]: Sesuai sample SRS
    Role        string   `json:"role"`
    Permissions []string `json:"permissions"` // [PERBAIKAN]: Ditambahkan
}