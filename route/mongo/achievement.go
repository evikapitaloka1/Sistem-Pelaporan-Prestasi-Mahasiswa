package routes

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid" // Tetap dipakai untuk userID

	// Service Interface untuk RBAC & Achievement
	achievementService "uas/app/service/mongo"
	authService "uas/app/service/postgres"

	// Model untuk request
	models "uas/app/model/mongo"

	// Middleware
	mw "uas/middleware"
)

// Helper function to safely get UUID from fiber context locals (TETAP UUID UNTUK USER ID)
func getUserID(c *fiber.Ctx) (uuid.UUID, error) {
	val := c.Locals("userID")
	
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user ID not found or invalid type (expected uuid.UUID)")
	}
	return id, nil
}

// Helper function to safely get role from fiber context locals
func getRole(c *fiber.Ctx) (string, error) {
	val := c.Locals("role")
	
	role, ok := val.(string)
	if !ok || role == "" {
		return "", errors.New("role not found in context or invalid type")
	}
	return role, nil
}


// AchievementRoutes sekarang menerima 2 Service Instance
func AchievementRoutes(
	api fiber.Router,
	authSvc *authService.AuthService, 
	achievementSvc achievementService.AchievementService,
) {
	// Inisialisasi Middleware
	jwtMiddleware := mw.JWTMiddleware()
	rbacCreate := mw.RBACMiddleware("achievement:create", authSvc)
	rbacUpdate := mw.RBACMiddleware("achievement:update", authSvc)
	rbacDelete := mw.RBACMiddleware("achievement:delete", authSvc)
	rbacVerify := mw.RBACMiddleware("achievement:verify", authSvc)
	rbacView := mw.RBACMiddleware("achievement:read", authSvc)

	// Group route /achievements dengan JWT
	achievements := api.Group("/achievements", jwtMiddleware)

	// ----------------------
	// A. ENDPOINT UMUM (READ/HISTORY)
	// ----------------------
	achievements.Get("/", rbacView, func(c *fiber.Ctx) error {
		userID, err := getUserID(c) 
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		
		role, err := getRole(c)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		
		result, err := achievementSvc.ListAchievements(c.Context(), role, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	achievements.Get("/:id", rbacView, func(c *fiber.Ctx) error {
		// ✅ PERBAIKAN: Ambil ID sebagai string (MongoDB ObjectID)
		achievementID := c.Params("id")
		
		// Opsional: Validasi panjang ObjectID jika diperlukan (biasanya 24 karakter heksa)
		if len(achievementID) != 24 {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid achievement ID format (Expected 24-char ObjectID)"})
		}
		
		userID, err := getUserID(c)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		
		role, err := getRole(c)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		// ✅ PERBAIKAN: Ganti pemanggilan service dari uuid.UUID ke string
		// ASUMSI GetAchievementDetail sekarang menerima ID bertipe string
		result, err := achievementSvc.GetAchievementDetail(c.Context(), achievementID, userID, role)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	achievements.Get("/:id/history", rbacView, func(c *fiber.Ctx) error {
		return c.SendString("Achievement History Endpoint")
	})

	// ----------------------
	// B. MAHASISWA (CREATE, UPDATE, DELETE)
	// ----------------------
	achievements.Post("/", rbacCreate, func(c *fiber.Ctx) error {
		var req models.AchievementRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body", "details": err.Error()})
		}
		
		userID, err := getUserID(c)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		
		result, err := achievementSvc.CreateAchievement(c.Context(), userID, req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(http.StatusCreated).JSON(fiber.Map{"status": "success", "data": result})
	})

	achievements.Put("/:id", rbacUpdate, func(c *fiber.Ctx) error {
		return c.SendString("Update Achievement Endpoint")
	})

	achievements.Delete("/:id", rbacDelete, func(c *fiber.Ctx) error {
		// ✅ PERBAIKAN: Ambil ID sebagai string (MongoDB ObjectID)
		achievementID := c.Params("id")
		if len(achievementID) != 24 {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid achievement ID format (Expected 24-char ObjectID)"})
		}

		userID, err := getUserID(c)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		// ✅ PERBAIKAN: Ganti pemanggilan service dari uuid.UUID ke string
		err = achievementSvc.DeleteAchievement(c.Context(), achievementID, userID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "message": "Achievement deleted"})
	})

	// ----------------------
	// C. DOSEN WALI (VERIFY, REJECT)
	// ----------------------
	achievements.Post("/:id/verify", rbacVerify, func(c *fiber.Ctx) error {
		// ✅ PERBAIKAN: Ambil ID sebagai string (MongoDB ObjectID)
		achievementID := c.Params("id")
		if len(achievementID) != 24 {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid achievement ID format (Expected 24-char ObjectID)"})
		}

		lecturerID, err := getUserID(c)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		// ✅ PERBAIKAN: Ganti pemanggilan service dari uuid.UUID ke string
		err = achievementSvc.VerifyAchievement(c.Context(), achievementID, lecturerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "message": "Achievement verified"})
	})

	achievements.Post("/:id/reject", rbacVerify, func(c *fiber.Ctx) error {
		// ✅ PERBAIKAN: Ambil ID sebagai string (MongoDB ObjectID)
		achievementID := c.Params("id")
		if len(achievementID) != 24 {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid achievement ID format (Expected 24-char ObjectID)"})
		}

		lecturerID, err := getUserID(c)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		
		type rejectReq struct {
			Note string `json:"note"`
		}
		var req rejectReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		// ✅ PERBAIKAN: Ganti pemanggilan service dari uuid.UUID ke string
		err = achievementSvc.RejectAchievement(c.Context(), achievementID, lecturerID, req.Note)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "message": "Achievement rejected"})
	})
}