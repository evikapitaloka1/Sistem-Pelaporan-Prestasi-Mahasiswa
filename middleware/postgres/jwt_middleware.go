package middleware

import (
	"strings"
	"uas/app/model/postgres"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret = []byte("SECRET_KEY")

func JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authorization header required"})
		}

		tokenStr := strings.Replace(authHeader, "Bearer ", "", 1)
		token, err := jwt.ParseWithClaims(tokenStr, &model.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token tidak valid"})
		}

		claims, ok := token.Claims.(*model.UserClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "claims token tidak valid"})
		}

		c.Locals("userID", claims.UserID)
		c.Locals("role", claims.Role)
		c.Locals("permissions", claims.Permissions)
		return c.Next()
	}
}
