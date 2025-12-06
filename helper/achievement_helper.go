package helper

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type BlacklistChecker interface {
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// Noop Blacklist Checker
type NoopBlacklistChecker struct{}

func (n *NoopBlacklistChecker) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

// Helper mengambil userID & role dari JWT
func GetUserData(c *fiber.Ctx) (uuid.UUID, string, error) {
	rawID := c.Locals("userID")

	userIDStr, ok := rawID.(string)
	if !ok || userIDStr == "" {
		return uuid.Nil, "", fiber.NewError(http.StatusUnauthorized,
			"userID tidak ditemukan atau format salah")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, "", fiber.NewError(http.StatusUnauthorized,
			"userID bukan UUID valid")
	}

	rawRole := c.Locals("role")
	role, okRole := rawRole.(string)
	if !okRole || role == "" {
		return uuid.Nil, "", fiber.NewError(http.StatusUnauthorized,
			"Role tidak ditemukan")
	}

	return userID, role, nil
}