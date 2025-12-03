package service

import (
	"context"
	"errors"
	"time"
	"os" // Diperlukan untuk os.Getenv

	"uas/app/model/postgres"
	"uas/app/repository/postgres"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Helper untuk mendapatkan Secret Key dari ENV
func getSecret() ([]byte, error) {
	secret := os.Getenv("SECRET_KEY")
	if secret == "" {
		// Asumsi main.go sudah memastikan SECRET_KEY ada, ini adalah fallback
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
	jwtSecret, err := getSecret() // ✅ Secret key diambil dari ENV
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

	// ✅ FIX JTI: Tambahkan ID (JTI) ke claims token akses
	accessJTI := uuid.New().String()
	
	claims := &model.UserClaims{
		UserID: 		user.ID,
		Username: 	user.Username,
		Role: 			user.Role,
		Permissions: user.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt: 	jwt.NewNumericDate(time.Now()),
			ID: 				accessJTI, // <<-- JTI DITAMBAHKAN DI SINI (Fix Logout)
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	// ✅ FIX JTI: Refresh token TIDAK menggunakan JTI yang sama (dikosongkan)
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
	jwtSecret, err := getSecret() // ✅ Secret key diambil dari ENV
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

	// ✅ FIX JTI: Tambahkan ID (JTI) ke klaim token baru
	accessJTI := uuid.New().String()
	
	newClaims := &model.UserClaims{
		UserID: 		 claims.UserID,
		Username: 	 claims.Username,
		Role: 			 claims.Role,
		Permissions: claims.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt: 	 jwt.NewNumericDate(time.Now()),
			ID: 				 accessJTI, // <<-- JTI DITAMBAHKAN DI SINI
		},
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return newToken.SignedString(jwtSecret)
}

// ================= LOGOUT =================
func (s *AuthService) Logout(ctx context.Context, jti string) error {
	// ✅ FIX LOGOUT: Gunakan repository untuk mem-blacklist JTI
	if jti == "" {
		return errors.New("JTI tidak boleh kosong")
	}
	// Memanggil repository untuk menyimpan JTI ke database/cache (blacklist)
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