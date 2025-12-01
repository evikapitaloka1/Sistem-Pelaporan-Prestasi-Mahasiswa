package middleware

import (
	"fmt"
	"strings" // Import strings untuk normalisasi role (walaupun di JWTMiddleware sudah dilakukan)
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

		// 🛑 PENTING: Role sudah disetel sebagai lowercase oleh JWTMiddleware
		// Kita lakukan normalisasi ulang di sini sebagai double-check yang aman.
		normalizedRole := strings.ToLower(role)

		if !ok || normalizedRole == "" {
			// Pengamanan jika role tidak disetel/hilang/tipe salah
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	401,
				"error": "Role is missing from context. Please login again.",
			})
		}

		// 2. ✅ LOGIKA BYPASS ADMIN (KOREKSI)
		// Sekarang cek role menggunakan huruf kecil
		if normalizedRole == "admin" { // <-- KOREKSI UTAMA: Cek "admin" (huruf kecil)
			return c.Next() // Admin selalu diizinkan, lewati pengecekan database
		}
		
		// 3. Ambil userID dari JWT middleware (hanya untuk Mahasiswa/Dosen Wali)
		userIDVal := c.Locals("userID")
		
		if userIDVal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 	401,
				"error": "User belum terautentikasi (Missing JWT claims)",
			})
		}

		// Pastikan type-nya uuid.UUID
		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code": 	500,
				"error": "Internal Server Error: User ID type assertion failed",
			})
		}

		// 4. Cek permission menggunakan Service (Postgres)
		hasPerm, err := authService.HasPermission(c.Context(), userID, permission)
		
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code": 	500,
				"error": fmt.Sprintf("Gagal cek permission: %v", err),
			})
		}

		if !hasPerm {
			// Mengembalikan error 403 Forbidden
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code": 	403,
				"error": fmt.Sprintf("Akses ditolak: Tidak memiliki izin '%s'", permission),
			})
		}

		// 5. Lanjut ke handler berikutnya (hanya jika role bukan Admin dan memiliki izin)
		return c.Next()
	}
}