package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// CustomClaims mendefinisikan struktur claim yang diharapkan dalam JWT.
type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"` // Ini akan berisi "Mahasiswa Pelapor"
	jwt.RegisteredClaims
}

// JWTMiddleware memverifikasi token JWT dari header Authorization.
func JWTMiddleware() fiber.Handler {
	// GANTI DENGAN SECRET KEY ASLI ANDA
	var jwtSecret = []byte("SECRET_KEY")
	
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	401,
				"error": "Akses ditolak: Missing Authorization Header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	401,
				"error": "Akses ditolak: Format token tidak valid (Harus: Bearer <token>)",
			})
		}

		tokenString := parts[1]
		claims := &CustomClaims{}
		
		// Parsing dan Validasi Token
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	401,
				"error": fmt.Sprintf("Token tidak valid: %v", err),
			})
		}
		
		// --- PERBAIKAN UTAMA: ROLE MAPPING DAN NORMALISASI ---
		rawRole := strings.ToLower(claims.Role) // rawRole = "mahasiswa pelapor"

		// 1. Lakukan pemetaan string kompleks ke string sederhana
		var standardizedRole string
		
		if strings.Contains(rawRole, "admin") {
			standardizedRole = "admin"
		} else if strings.Contains(rawRole, "dosen wali") || strings.Contains(rawRole, "verifikator") {
			standardizedRole = "dosen wali"
		} else if strings.Contains(rawRole, "mahasiswa") || strings.Contains(rawRole, "pelapor") {
			standardizedRole = "mahasiswa"
		} else {
            // Jika role tidak dikenali, set ke string kosong atau error
            return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "code": 403,
                "error": fmt.Sprintf("Role '%s' tidak dikenali oleh sistem prestasi.", claims.Role),
            })
        }
		
		// 2. Parsing user_id (string) menjadi uuid.UUID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	401,
				"error": "User ID dalam token bukan format UUID yang valid",
			})
		}

		// 3. Simpan data yang sudah distandardisasi
		c.Locals("userID", userID) 
		c.Locals("role", standardizedRole) // <-- Disimpan sebagai "mahasiswa", "admin", atau "dosen wali"
		
		return c.Next()
	}
}