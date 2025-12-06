package helper

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Mengambil UserID (UUID) dari JWT Middleware
func GetUserIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	userLocals := c.Locals("userID")

	// Coba string
	if str, ok := userLocals.(string); ok && str != "" {
		return uuid.Parse(str)
	}

	// Coba UUID langsung
	if uid, ok := userLocals.(uuid.UUID); ok {
		return uid, nil
	}

	return uuid.Nil, errors.New("user ID missing or invalid type in token")
}

// Mengambil role dari token
func GetRoleFromContext(c *fiber.Ctx) (string, error) {
	role, ok := c.Locals("role").(string)
	if !ok || role == "" {
		return "", errors.New("user role missing from token")
	}
	return strings.ToLower(role), nil
}

// Helper balikan error standard JSON
func JsonError(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// Cek unauthorized dari error service layer
func IsUnauthorizedErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unauthorized")
}
