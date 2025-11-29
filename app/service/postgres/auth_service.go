package service

import (
	"context"
	"errors"
	"time"

	"uas/app/model/postgres"
	"uas/app/repository/postgres"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
)

var jwtSecret = []byte("SECRET_KEY")

type AuthService struct {
	repo repository.AuthRepository
}

// Constructor
func NewAuthService(repo repository.AuthRepository) *AuthService {
	return &AuthService{repo: repo}
}

// ================= LOGIN =================
func (s *AuthService) Login(ctx context.Context, username string, password string) (*model.LoginResponse, error) {
	user, hash, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// cek password
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, errors.New("password salah")
	}

	// buat JWT claims
	claims := &model.UserClaims{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: user.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	// generate refresh token (misal 24 jam)
	refreshClaims := *claims
	refreshClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	// format response sesuai request
	resp := &model.LoginResponse{
		Status: "success",
	}
	resp.Data.Token = tokenString
	resp.Data.RefreshToken = refreshTokenString
	resp.Data.User = *user

	return resp, nil
}

// ================= REFRESH =================
func (s *AuthService) Refresh(ctx context.Context, tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return "", errors.New("token tidak valid")
	}

	claims, ok := token.Claims.(*model.UserClaims)
	if !ok || !token.Valid {
		return "", errors.New("claims token tidak valid")
	}

	newClaims := &model.UserClaims{
		UserID:      claims.UserID,
		Username:    claims.Username,
		Role:        claims.Role,
		Permissions: claims.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	newTokenString, err := newToken.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return newTokenString, nil
}

// ================= LOGOUT =================
func (s *AuthService) Logout(ctx context.Context) error {
	// Stateless, tidak perlu simpan token
	return nil
}
// ================= PROFILE =================
func (s *AuthService) Profile(ctx context.Context, userID uuid.UUID) (*model.UserData, error) {
    user, err := s.repo.GetByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    return user, nil
}

