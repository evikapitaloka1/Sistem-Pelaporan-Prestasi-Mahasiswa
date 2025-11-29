package utils

import (
	"time"
	"uas/app/model/postgres"

	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret = []byte("SECRET_KEY")

// GenerateToken buat JWT baru
func GenerateToken(user *model.UserData) (string, string, error) {
	claims := &model.UserClaims{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: user.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)), // token 1 jam
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// access token
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := tokenObj.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	// refresh token (bisa berbeda durasi kalau mau)
	refreshObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refreshToken, err := refreshObj.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshToken, nil
}

// ParseToken memvalidasi token dan mengembalikan claims
func ParseToken(tokenString string) (*model.UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*model.UserClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
