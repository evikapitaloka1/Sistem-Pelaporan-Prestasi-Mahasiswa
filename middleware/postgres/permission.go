package middleware

import (
	
	"uas/app/service/postgres"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// PermissionMiddleware ngecek permission user
func PermissionMiddleware(permission string, authService service.IAuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ambil userID dari JWT middleware
		userIDVal := c.Locals("userID")
		if userIDVal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "user belum login",
			})
		}

		// pastikan type-nya uuid.UUID
		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "userID tidak valid",
			})
		}

		// cek permission
		hasPerm, err := authService.HasPermission(c.Context(), userID, permission)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if !hasPerm {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "akses ditolak",
			})
		}

		// lanjut ke handler berikutnya
		return c.Next()
	}
}
