package routes

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	model "uas/app/model/postgres"
	service "uas/app/service/postgres"
	"uas/helper"
)

func SetupAuthRoutes(router fiber.Router, authService *service.AuthService, jwtMiddleware fiber.Handler) {

	auth := router.Group("/auth")

	// ================= LOGIN =================
	auth.Post("/login", func(c *fiber.Ctx) error {

		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.SendError(c, fiber.StatusBadRequest, "Invalid request body")
		}

		resp, err := authService.Login(c.Context(), req.Username, req.Password)
		if err != nil {
			return helper.SendError(c, fiber.StatusUnauthorized, err.Error())
		}

		return helper.SendSuccess(c, resp)
	})

	// ================= REFRESH =================
	auth.Post("/refresh", func(c *fiber.Ctx) error {

		var refreshToken string

		// Header: Authorization: Bearer <token>
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			refreshToken = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// Body
		if refreshToken == "" {
			var payload struct {
				RefreshToken string `json:"refresh_token"`
			}
			_ = c.BodyParser(&payload)
			if payload.RefreshToken != "" {
				refreshToken = payload.RefreshToken
			}
		}

		// Tidak dapat refresh token
		if refreshToken == "" {
			return helper.SendError(c, fiber.StatusUnauthorized, "Refresh token missing")
		}

		token, err := authService.Refresh(c.Context(), refreshToken)
		if err != nil {
			return helper.SendError(c, fiber.StatusUnauthorized, err.Error())
		}

		return helper.SendSuccess(c, fiber.Map{"token": token})
	})

	// ================= LOGOUT =================
	auth.Post("/logout", jwtMiddleware, func(c *fiber.Ctx) error {

		jti, ok := c.Locals("jti").(string)
		if !ok || jti == "" {
			return helper.SendError(c, fiber.StatusUnauthorized, "Missing JTI in token")
		}

		if err := authService.Logout(c.Context(), jti); err != nil {
			return helper.SendError(c, fiber.StatusInternalServerError, err.Error())
		}

		return helper.SendSuccess(c, "logout success")
	})

	// ================= PROFILE =================
	auth.Get("/profile", jwtMiddleware, func(c *fiber.Ctx) error {
    userIDLocal := c.Locals("userID")
    userIDStr, ok := userIDLocal.(string)
    if !ok || userIDStr == "" {
        return helper.SendError(c, fiber.StatusUnauthorized, "Invalid or missing user ID in token")
    }

    // Parse ke UUID
    uid, err := uuid.Parse(userIDStr)
    if err != nil {
        return helper.SendError(c, fiber.StatusUnauthorized, "Invalid user ID format in token")
    }

    user, err := authService.Profile(c.Context(), uid)
    if err != nil {
        return helper.SendError(c, fiber.StatusNotFound, err.Error())
    }

    return helper.SendSuccess(c, user)
})

}
