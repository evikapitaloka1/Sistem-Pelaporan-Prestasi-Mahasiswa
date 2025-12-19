package service

import (
	"os"
	"strings"
	"time"

	"sistempelaporan/app/model"
	"sistempelaporan/app/repository"
	"sistempelaporan/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Helper: Ambil JWT Secret konsisten dari satu sumber
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("rahasia_negara_api")
	}
	return []byte(secret)
}

// Login godoc
// @Summary      Login Pengguna
// @Description  Otentikasi pengguna menggunakan username dan password untuk mendapatkan JWT Token.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        login  body      object  true  "Kredensial Login (Username & Password)"
// @Success      200    {object}  helper.Response{data=model.LoginResponse} // <-- PERBAIKAN DI SINI
// @Failure      401    {object}  helper.Response
// @Router       /auth/login [post]
func Login(c *fiber.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", nil)
	}

	user, err := repository.FindUserByUsername(input.Username)
	if err != nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Username atau password salah", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Username atau password salah", nil)
	}

	perms, _ := repository.GetPermissionsByRoleID(user.RoleID.String())

	if strings.EqualFold(user.Role.Name, "Admin") {
		hasReadAll := false
		for _, p := range perms {
			if p == "user:read_all" {
				hasReadAll = true
				break
			}
		}
		if !hasReadAll {
			perms = append(perms, "user:read_all")
		}
	}

	accessToken, refreshToken, err := generateTokens(user, perms)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal generate token", nil)
	}

	response := model.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User: model.UserResponse{
			ID:          user.ID.String(),
			Username:    user.Username,
			FullName:    user.FullName,
			Role:        user.Role.Name,
			Permissions: perms,
		},
	}

	return helper.Success(c, response, "Login berhasil")
}

// RefreshToken godoc
// @Summary      Perbarui Token
// @Description  Memperbarui Access Token yang kadaluarsa menggunakan Refresh Token yang valid.
// @Tags         Authentication
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer {refresh_token}"
// @Success      200            {object}  model.LoginResponse
// @Failure      401            {object}  helper.Response
// @Router       /auth/refresh [post]
// @Security     BearerAuth
func RefreshToken(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return helper.Error(c, fiber.StatusUnauthorized, "Missing Authorization header for refresh token", nil)
	}
	refreshTokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	if refreshTokenString == "" {
		return helper.Error(c, fiber.StatusUnauthorized, "Refresh token is empty", nil)
	}

	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	if err != nil || !token.Valid {
		return helper.Error(c, fiber.StatusUnauthorized, "Refresh token tidak valid atau kadaluarsa", err.Error())
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Token claims invalid", nil)
	}

	userIDStr, _ := claims["user_id"].(string)

	user, err := repository.FindUserByID(userIDStr)
	if err != nil {
		return helper.Error(c, fiber.StatusUnauthorized, "User tidak ditemukan", nil)
	}
	
	perms, _ := repository.GetPermissionsByRoleID(user.RoleID.String())
	
	newAccess, newRefresh, err := generateTokens(user, perms)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal refresh token", err.Error())
	}

	return helper.Success(c, &model.LoginResponse{
		Token:        newAccess,
		RefreshToken: newRefresh,
		User: model.UserResponse{
			ID:          user.ID.String(),
			Username:    user.Username,
			FullName:    user.FullName,
			Role:        user.Role.Name,
			Permissions: perms,
		},
	}, "Token diperbarui")
}

// Logout godoc
// @Summary      Logout Pengguna
// @Description  Mengakhiri sesi pengguna dan memasukkan token ke dalam daftar blacklist.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  helper.Response
// @Router       /auth/logout [post]
// @Security     BearerAuth
func Logout(c *fiber.Ctx) error {
	tokenString := strings.Replace(c.Get("Authorization"), "Bearer ", "", 1)
	
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})
	
	claims, _ := token.Claims.(jwt.MapClaims)
	expTimeUnix := int64(claims["exp"].(float64))
	
	expiration := time.Unix(expTimeUnix, 0)
	ttl := expiration.Sub(time.Now())
	
	if err := repository.SetTokenBlacklist(tokenString, ttl); err != nil {
		// Log error
	}
	
	return helper.Success(c, nil, "Logout berhasil")
}

// GetProfile godoc
// @Summary      Dapatkan Profil Saya
// @Description  Mengambil informasi profil detail pengguna yang sedang login berdasarkan token.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  model.User
// @Router       /auth/profile [get]
// @Security     BearerAuth
func GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	user, err := repository.FindUserByID(userID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "User tidak ditemukan", err.Error())
	}
	return helper.Success(c, user, "Profil user")
}

func generateTokens(user *model.User, perms []string) (string, string, error) {
	secret := getJWTSecret()

	claims := jwt.MapClaims{
		"user_id":     user.ID,
		"role":        user.Role.Name,
		"permissions": perms,
		"exp":         time.Now().Add(2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(secret)
	if err != nil { return "", "", err }

	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	rt, err := refreshToken.SignedString(secret)

	return t, rt, err
}