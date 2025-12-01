package routes

import (
	"uas/app/model/postgres"
	service "uas/app/service/postgres"
	
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func SetupAuthRoutes(router fiber.Router, authService *service.AuthService) {

	auth := router.Group("/auth")

	// ================= LOGIN =================
	auth.Post("/login", func(c *fiber.Ctx) error {
		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		resp, err := authService.Login(c.Context(), req.Username, req.Password)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"data": resp})
	})

	// ================= REFRESH =================
	auth.Post("/refresh", func(c *fiber.Ctx) error {
		var payload struct {
			RefreshToken string `json:"refresh_token"`
		}

		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		token, err := authService.Refresh(c.Context(), payload.RefreshToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"token": token})
	})

	// ================= LOGOUT =================
	auth.Post("/logout", func(c *fiber.Ctx) error {
		err := authService.Logout(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "logout success"})
	})

	// ================= PROFILE =================
	auth.Get("/profile", func(c *fiber.Ctx) error {

		userIDHeader := c.Get("X-User-ID")
		if userIDHeader == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing user id"})
		}

		uid, err := uuid.Parse(userIDHeader)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
		}

		user, err := authService.Profile(c.Context(), uid)
		if err != nil {
			
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"data": user})
	})
}
