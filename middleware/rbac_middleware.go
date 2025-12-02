package middleware

import (
	"fmt"
	"strings" 
	service "uas/app/service/postgres" 

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RBACMiddleware memverifikasi apakah user memiliki izin yang diperlukan.
func RBACMiddleware(permission string, authService service.IAuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		
		// 1. Ambil Role dari JWT middleware
		roleVal := c.Locals("role")
		role, ok := roleVal.(string)

		normalizedRole := strings.ToLower(role)

		if !ok || normalizedRole == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	 401,
				"error": "Role is missing from context. Please login again.",
			})
		}

		// 2. LOGIKA BYPASS ADMIN
		if normalizedRole == "admin" {
			return c.Next() // Admin selalu diizinkan
		}
		
		// 3. Ambil dan Parse userID dari JWT middleware 
		
        // 🛑 KOREKSI UTAMA 1: Ambil sebagai STRING
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr || userIDStr == "" {
            // Ini akan menangani error "User ID missing or invalid from token"
            // jika token tidak di-set sama sekali
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	 401,
				"error": "User ID missing from context (Must be string).",
			})
		}
        
        // 🛑 KOREKSI UTAMA 2: Parse STRING menjadi UUID
        userID, err := uuid.Parse(userIDStr)
        if err != nil {
             return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code": 	 500,
				"error": "Internal Server Error: User ID in context is not a valid UUID format.",
			})
        }
        
        // 🛑 KOREKSI UTAMA 3: Set kembali userID ke locals sebagai UUID 
        //                    agar service bisa menggunakannya (Opsional, tapi membantu)
        c.Locals("parsedUserID", userID) // Gunakan key baru atau timpa jika JWT menyimpannya sebagai string

		// 4. Cek permission menggunakan Service (Postgres)
		hasPerm, err := authService.HasPermission(c.Context(), userID, permission)
		
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code": 	 500,
				"error": fmt.Sprintf("Gagal cek permission: %v", err),
			})
		}

		if !hasPerm {
			// Mengembalikan error 403 Forbidden
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code": 	 403,
				"error": fmt.Sprintf("Akses ditolak: Tidak memiliki izin '%s'", permission),
			})
		}

		// 5. Lanjut ke handler berikutnya 
		return c.Next()
	}
}