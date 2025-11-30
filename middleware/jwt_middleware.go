package middleware

import (

	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4" // Menggunakan v5 agar konsisten
	"github.com/google/uuid"
)

// CustomClaims mendefinisikan struktur claim yang diharapkan dalam JWT.
type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTMiddleware memverifikasi token JWT dari header Authorization.
func JWTMiddleware() fiber.Handler {
	// 🎯 PERBAIKAN: GANTI DENGAN SECRET KEY ASLI ANDA
	// Pastikan nilai ini SAMA PERSIS dengan secret yang digunakan saat membuat token.
	var jwtSecret = []byte("SECRET_KEY")
	
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"error": "Akses ditolak: Missing Authorization Header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"error": "Akses ditolak: Format token tidak valid (Harus: Bearer <token>)",
			})
		}

		tokenString := parts[1]
		claims := &CustomClaims{}
        
        // Perbaikan: Menggunakan jwt.ParseWithClaims dari v5
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
            // Cek apakah error karena token kadaluarsa atau signature invalid
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"error": fmt.Sprintf("Token tidak valid: %v", err),
			})
		}
        
		// 🎯 PERBAIKAN: Parsing user_id (string) menjadi uuid.UUID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"error": "User ID dalam token bukan format UUID yang valid",
			})
		}

		// 🎯 Simpan userID sebagai uuid.UUID (Memperbaiki Panic)
		c.Locals("userID", userID) 
		// Simpan role untuk Service Layer
		c.Locals("role", claims.Role)
		
		return c.Next()
	}
}