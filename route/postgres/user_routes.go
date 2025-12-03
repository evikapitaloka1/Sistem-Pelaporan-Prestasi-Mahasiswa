package routes

import (
	postgres"uas/app/service/postgres"
	mw "uas/middleware" // 🛑 PERBAIKAN 1: Import package middleware dengan alias 'mw'
	
	// Catatan: Interface IUserService dan IAuthService diasumsikan didefinisikan di service "uas/app/service/postgres"
	// Catatan: model.CreateUserRequest dll. diasumsikan didefinisikan di "uas/app/model/postgres"
	
	"uas/app/model/postgres"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// 🛑 PERBAIKAN SIGNATURE: Menerima jwtMiddleware yang sudah siap dari RegisterRoutes
func SetupUserRoutes(
	router fiber.Router,
	userService postgres.IUserService,
	authService postgres.IAuthService,
	jwtMiddleware fiber.Handler, // 🛑 Menerima JWT Handler yang sudah disuntikkan
) {
	// Inisialisasi RBAC Middleware (Asumsi PermissionMiddleware Anda adalah RBACMiddleware)
	rbacManage := mw.RBACMiddleware("user:manage", authService)
	// rbacRead := mw.RBACMiddleware("user:read", authService) // Tambahkan jika perlu

	// 🛑 PERBAIKAN 2: Gunakan jwtMiddleware yang di-inject. Tidak perlu memanggil JWTMiddleware lagi.
	// users := router.Group("/users", mw.JWTMiddleware(blacklistChecker)) // <--- BARIS LAMA
	users := router.Group("/users", jwtMiddleware)

	// GET all users (hanya yang punya permission "user:manage")
	// 🛑 PERBAIKAN 3: Ganti middleware.PermissionMiddleware menjadi rbacManage
	users.Get("/", rbacManage, func(c *fiber.Ctx) error {
		list, err := userService.GetAllUsers(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(list)
	})

	// GET user by ID
	users.Get("/:id", rbacManage, func(c *fiber.Ctx) error {
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
	users.Post("/", rbacManage, func(c *fiber.Ctx) error {
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
	users.Put("/:id", rbacManage, func(c *fiber.Ctx) error {
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
	users.Delete("/:id", rbacManage, func(c *fiber.Ctx) error {
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