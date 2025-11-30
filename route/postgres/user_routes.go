package routes

import (
	"uas/app/service/postgres"
	"uas/middleware"
	"uas/app/model/postgres"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func SetupUserRoutes(
	router fiber.Router,
	userService service.IUserService,
	authService service.IAuthService,
) {
	users := router.Group("/users", middleware.JWTMiddleware())

	// GET all users (hanya yang punya permission "user:manage")
	users.Get("/", middleware.PermissionMiddleware("user:manage", authService), func(c *fiber.Ctx) error {
		list, err := userService.GetAllUsers(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(list)
	})

	// GET user by ID
	users.Get("/:id", middleware.PermissionMiddleware("user:manage", authService), func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		userID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
		}

		user, err := userService.GetUserByID(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
		}
		return c.JSON(user)
	})

	// POST create user
	users.Post("/", middleware.PermissionMiddleware("user:manage", authService), func(c *fiber.Ctx) error {
		var req model.CreateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		newID, err := userService.CreateUser(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"id": newID})
	})

	// PUT update user
	users.Put("/:id", middleware.PermissionMiddleware("user:manage", authService), func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		userID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
		}

		var req model.UpdateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if err := userService.UpdateUser(c.Context(), userID, req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	// DELETE user
	users.Delete("/:id", middleware.PermissionMiddleware("user:manage", authService), func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		userID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
		}

		if err := userService.DeleteUser(c.Context(), userID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(fiber.StatusOK)
	})
}
