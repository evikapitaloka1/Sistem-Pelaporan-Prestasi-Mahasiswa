package service

import (
    "os"
    "time"
    "strings"
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

// 1. Login
func Login(c *fiber.Ctx) error {
	// 1. Parsing Request
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", nil)
	}

	// 2. Cari User berdasarkan Username
	user, err := repository.FindUserByUsername(input.Username)
	if err != nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Username atau password salah", nil)
	}

	// 3. Validasi Password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Username atau password salah", nil)
	}

	// 4. Ambil Permissions dari Database
	perms, _ := repository.GetPermissionsByRoleID(user.RoleID.String())

	// --- PERBAIKAN: JAMIN ADMIN MEMILIKI HAK "user:read_all" ---
	if strings.EqualFold(user.Role.Name, "Admin") {
		hasReadAll := false
		for _, p := range perms {
			if p == "user:read_all" {
				hasReadAll = true
				break
			}
		}
		
		// Jika Admin tidak punya permission ini di DB, tambahkan secara paksa ke list
		if !hasReadAll {
			perms = append(perms, "user:read_all")
		}
		
		// Anda bisa menambahkan permission krusial lain di sini jika dibutuhkan
		// perms = append(perms, "user:manage")
	}

	// 5. Generate Token dengan list permissions yang sudah fix
	accessToken, refreshToken, err := generateTokens(user, perms)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal generate token", nil)
	}

	// 6. Return Response (Sesuai model response)
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
func RefreshToken(c *fiber.Ctx) error {
	// 1. Ambil Refresh Token string dari Header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return helper.Error(c, fiber.StatusUnauthorized, "Missing Authorization header for refresh token", nil)
	}
	refreshTokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	if refreshTokenString == "" {
		return helper.Error(c, fiber.StatusUnauthorized, "Refresh token is empty", nil)
	}

	// 2. Parse & Validasi Refresh Token
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	if err != nil || !token.Valid {
		return helper.Error(c, fiber.StatusUnauthorized, "Refresh token tidak valid atau kadaluarsa", err.Error())
	}

	// 3. Ambil User ID dari klaim
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Token claims invalid", nil)
	}

	userIDStr, _ := claims["user_id"].(string)

	user, err := repository.FindUserByID(userIDStr)
	if err != nil {
		return helper.Error(c, fiber.StatusUnauthorized, "User tidak ditemukan", nil)
	}
    // FIX 2.1: Ganti .(string) menjadi .String()
	perms, _ := repository.GetPermissionsByRoleID(user.RoleID.String())
	
	// 4. Generate Token Baru
	newAccess, newRefresh, err := generateTokens(user, perms)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal refresh token", err.Error())
	}

	return helper.Success(c, &model.LoginResponse{
		Token: newAccess,
		RefreshToken: newRefresh,
		User: model.UserResponse{
            // FIX 2.2: Ganti .(string) menjadi .String()
			ID: user.ID.String(),
			Username: user.Username,
			FullName: user.FullName,
			Role: user.Role.Name,
			Permissions: perms,
		},
	}, "Token diperbarui")
}
// ... (Sisa fungsi Logout, GetProfile, dan generateTokens biarkan SAMA) ...
// Copy paste saja fungsi generateTokens, Logout, GetProfile dari jawaban sebelumnya.
// Perubahan HANYA pada penambahan .String() di dua baris di atas.

// 3. Logout
// import "time" dan package redis (asumsi Anda menggunakan Redis/Cache)
// Misalnya kita asumsikan ada package cache yang punya fungsi SetBlacklist

// 3. Logout (Perlu Blacklist)
func Logout(c *fiber.Ctx) error {
    // 1. Ambil token dari header (Middleware Protected() sudah memastikan token ada)
    tokenString := strings.Replace(c.Get("Authorization"), "Bearer ", "", 1)
    
    // 2. Parse token untuk mendapatkan klaim 'exp'
    token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return getJWTSecret(), nil
    })
    
    claims, _ := token.Claims.(jwt.MapClaims)
    expTimeUnix := int64(claims["exp"].(float64)) // Ambil waktu kadaluarsa (Unix Timestamp)
    
    // 3. Hitung sisa waktu token (TTL)
    expiration := time.Unix(expTimeUnix, 0)
    ttl := expiration.Sub(time.Now())
    
    // 4. Masukkan token ke Blacklist dengan TTL
    // ASUMSI: repository.SetTokenBlacklist(token, ttl) menyimpan tokenString di Redis
    if err := repository.SetTokenBlacklist(tokenString, ttl); err != nil {
        // Log error, tapi tetap sukseskan logout
    }

    // [OPSIONAL]: Blacklist Refresh Token juga
    // ...
    
    return helper.Success(c, nil, "Logout berhasil")
}

// 4. Get Profile
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

	// Access Token
	claims := jwt.MapClaims{
		"user_id":     user.ID,
		"role":        user.Role.Name,
		"permissions": perms,
		"exp":         time.Now().Add(2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(secret)
	if err != nil { return "", "", err }

	// Refresh Token
	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	rt, err := refreshToken.SignedString(secret)

	return t, rt, err
}