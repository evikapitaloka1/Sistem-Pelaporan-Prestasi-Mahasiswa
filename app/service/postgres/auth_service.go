package services

import (
	"context"
	"errors"
	"time"
	"os"

	"uas/app/model/postgres"
	"uas/app/repository/postgres"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func getSecret() ([]byte, error) {
	secret := os.Getenv("SECRET_KEY")
	if secret == "" {
		return nil, errors.New("konfigurasi error: SECRET_KEY tidak ditemukan")
	}
	return []byte(secret), nil
}

// ================= INTERFACE =================
type IAuthService interface {
	Login(ctx context.Context, username, password string) (*model.LoginResponse, error)
	Refresh(ctx context.Context, tokenString string) (string, error)
	Logout(ctx context.Context, jti string) error
	Profile(ctx context.Context, userID uuid.UUID) (*model.UserData, error)
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}

// ================= STRUCT =================
type AuthService struct {
	repo repository.AuthRepository
}

var _ IAuthService = &AuthService{}

// ================= CONSTRUCTOR =================
func NewAuthService(repo repository.AuthRepository) *AuthService {
	return &AuthService{repo: repo}
}

// ================= LOGIN =================
func (s *AuthService) Login(ctx context.Context, username, password string) (*model.LoginResponse, error) {
	jwtSecret, err := getSecret()
	if err != nil {
		return nil, err
	}

	user, hash, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, errors.New("password salah")
	}

	accessJTI := uuid.New().String()

	claims := &model.UserClaims{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: user.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        accessJTI,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshClaims := *claims
	refreshClaims.RegisteredClaims.ID = ""
	refreshClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24))

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, &refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	resp := &model.LoginResponse{
		Status: "success",
	}
	resp.Data.Token = tokenString
	resp.Data.RefreshToken = refreshTokenString
	resp.Data.User = *user

	return resp, nil
}

// ================= REFRESH TOKEN =================
func (s *AuthService) Refresh(ctx context.Context, tokenString string) (string, error) {
	jwtSecret, err := getSecret()
	if err != nil {
		return "", err
	}

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

	accessJTI := uuid.New().String()

	newClaims := &model.UserClaims{
		UserID:      claims.UserID,
		Username:    claims.Username,
		Role:        claims.Role,
		Permissions: claims.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        accessJTI,
		},
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return newToken.SignedString(jwtSecret)
}

// ================= LOGOUT =================
func (s *AuthService) Logout(ctx context.Context, jti string) error {
	if jti == "" {
		return errors.New("JTI tidak boleh kosong")
	}
	return s.repo.BlacklistToken(ctx, jti)
}

// ================= PROFILE =================
func (s *AuthService) Profile(ctx context.Context, userID uuid.UUID) (*model.UserData, error) {
	return s.repo.GetByID(ctx, userID)
}

// ================= HAS PERMISSION =================
func (s *AuthService) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, p := range user.Permissions {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}