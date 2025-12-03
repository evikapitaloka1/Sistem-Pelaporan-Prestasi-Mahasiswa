package middleware

import (
	"fmt"
	"strings" 
	service "uas/app/service/postgres" 
	 
	// ✅ FIX: Tambahkan import UUID karena HasPermission menggunakannya
	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"

)

// RBACMiddleware memverifikasi apakah user memiliki izin yang diperlukan.
// PENTING: Middleware ini sekarang menggunakan HasPermission (User-centric) karena
// GetPermissionsByRole tidak tersedia.
func RBACMiddleware(permission string, authService service.IAuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		
		// 1. Ambil Role dari JWT middleware
		roleVal := c.Locals("role")
		role, ok := roleVal.(string)

		normalizedRole := strings.ToLower(role)

		if !ok || normalizedRole == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	401,
				"error": "Role is missing from context. Please login again.",
			})
		}

		// 2. LOGIKA BYPASS ADMIN (Menggunakan role 'admin' yang sudah dinormalisasi)
		if normalizedRole == "admin" {
			return c.Next() // Admin selalu diizinkan
		}
		
		// 3. PENGAMBILAN USER ID (Wajib untuk HasPermission)
		
		var userIDStr string
		userLocals := c.Locals("userID")
		
		// Ambil User ID sebagai string (tipe dari JWTMiddleware) atau UUID (jika tipe lama)
		if str, ok := userLocals.(string); ok {
			userIDStr = str
		} else if uid, ok := userLocals.(uuid.UUID); ok {
			userIDStr = uid.String()
		}

		if userIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	401,
				"error": "User ID missing or invalid type from token.",
			})
		}

		// Parse string ke UUID
		userID, err := uuid.Parse(userIDStr) 
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code": 	500,
				"error": "Internal Server Error: User ID in context is not a valid UUID format.",
			})
		}

		// 4. Cek permission menggunakan HasPermission (User-centric Check)
		// ✅ FIX: Memanggil HasPermission dengan 3 argumen yang sesuai
		hasPerm, err := authService.HasPermission(c.Context(), userID, permission)
		
		if err != nil {
			// Gagal menghubungi database
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code": 	500,
				"error": fmt.Sprintf("Gagal memeriksa izin: %v", err),
			})
		}
		
		if !hasPerm {
			// Mengembalikan error 403 Forbidden
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code": 	403,
				"error": fmt.Sprintf("Akses ditolak: Tidak memiliki izin '%s'", permission),
			})
		}
		
		// 5. Lanjut ke handler berikutnya (Lolos RBAC)
		return c.Next()
	}
}