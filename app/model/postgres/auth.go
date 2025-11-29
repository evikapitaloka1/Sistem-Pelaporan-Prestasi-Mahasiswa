package model

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// ================= REQUEST =================
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ================= RESPONSE =================
type UserData struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	FullName    string    `json:"fullName"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
}

type LoginResponse struct {
	Status string `json:"status"` // "success"
	Data   struct {
		Token        string   `json:"token"`
		RefreshToken string   `json:"refreshToken"`
		User         UserData `json:"user"`
	} `json:"data"`
}

// ================= JWT CLAIMS =================
type UserClaims struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	jwt.RegisteredClaims
}
