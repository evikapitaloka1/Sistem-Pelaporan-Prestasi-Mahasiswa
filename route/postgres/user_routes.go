package routes

import (
	"uas/app/model/postgres"
	service "uas/app/service/postgres"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func SetupUserRoutes(router fiber.Router, userService *service.UserService) {

	users := router.Group("/users")

	// ========== GET ALL USERS ==========
	users.Get("/", func(c *fiber.Ctx) error {

		list, err := userService.GetAllUsers(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				fiber.Map{"error": err.Error()},
			)
		}

		return c.JSON(list)
	})

	// ========== GET USER BY ID ==========
	users.Get("/:id", func(c *fiber.Ctx) error {

		uid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "invalid user id"},
			)
		}

		user, err := userService.GetUserByID(c.Context(), uid)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(
				fiber.Map{"error": err.Error()},
			)
		}

		return c.JSON(user)
	})

	// ========== CREATE USER ==========
	users.Post("/", func(c *fiber.Ctx) error {

		var req model.CreateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "invalid request body"},
			)
		}

		id, err := userService.CreateUser(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": err.Error()},
			)
		}

		return c.JSON(fiber.Map{"user_id": id})
	})

	// ========== UPDATE USER ==========
	users.Put("/:id", func(c *fiber.Ctx) error {

		uid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "invalid user id"},
			)
		}

		var req model.UpdateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "invalid request body"},
			)
		}

		if err := userService.UpdateUser(c.Context(), uid, req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": err.Error()},
			)
		}

		return c.JSON(fiber.Map{"message": "user updated"})
	})

	// ========== DELETE USER ==========
	users.Delete("/:id", func(c *fiber.Ctx) error {

		uid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "invalid user id"},
			)
		}

		if err := userService.DeleteUser(c.Context(), uid); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": err.Error()},
			)
		}

		return c.JSON(fiber.Map{"message": "user deleted"})
	})

	// ========== UPDATE USER ROLE ==========
	users.Put("/:id/role", func(c *fiber.Ctx) error {

		userID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "invalid user id"},
			)
		}

		var body struct {
			RoleID string `json:"role_id"`
		}

		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "invalid request body"},
			)
		}

		roleID, err := uuid.Parse(body.RoleID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "invalid role id"},
			)
		}

		if err := userService.UpdateUserRole(c.Context(), userID, roleID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": err.Error()},
			)
		}

		return c.JSON(fiber.Map{"message": "role updated"})
	})
}
