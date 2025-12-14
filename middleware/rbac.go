package middleware

import (
	"strings"

	"sistempelaporan/app/repository" // Perlu di-import untuk Blacklist

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Gunakan key yang SAMA dengan di auth_service.go
// Idealnya ambil dari os.Getenv("JWT_SECRET")
var jwtKey = []byte("rahasia_negara_api")

// =======================================================
// 1. AuthMiddleware: Memvalidasi Token JWT + Blacklist Check
// =======================================================
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Ambil Token
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Missing token"})
		}
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// 2. Parse & Validasi Tanda Tangan/Expiration
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
		}
		
		// --- TAMBAHAN PENTING: CEK BLACKLIST ---
		if repository.IsTokenBlacklisted(tokenString) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Token has been revoked (logged out)"})
		}
		// ----------------------------------------


		// 3. Ambil data (Claims) dari dalam token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid token claims"})
		}

		// 4. SIMPAN DATA USER KE CONTEXT (untuk service/controller)
		c.Locals("user_id", claims["user_id"].(string)) // Konversi ke string
		c.Locals("role", claims["role"])
		
		// Parsing permissions
		var permissions []string
		if permsRaw, ok := claims["permissions"].([]interface{}); ok {
			for _, p := range permsRaw {
				permissions = append(permissions, p.(string))
			}
		}
		c.Locals("permissions", permissions)

		return c.Next()
	}
}

// =======================================================
// 2. PermissionMiddleware: Mengecek hak akses (RBAC)
// =======================================================
func CheckPermission(requiredPerm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userPerms, ok := c.Locals("permissions").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "No permissions found"})
		}

		hasPermission := false
		for _, p := range userPerms {
			if p == requiredPerm {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Forbidden: You don't have permission " + requiredPerm,
			})
		}

		return c.Next()
	}
}