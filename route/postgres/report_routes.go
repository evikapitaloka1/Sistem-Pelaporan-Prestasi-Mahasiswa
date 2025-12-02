package routes

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	mongo "uas/app/service/mongo"
	authService "uas/app/service/postgres"
	mw "uas/middleware"
)

// ReportRoutes setup semua route terkait laporan
func ReportRoutes(
	api fiber.Router,
	authSvc *authService.AuthService, // Diperlukan untuk RBACMiddleware
	achievementSvc mongo.AchievementService,
) {
	jwtMiddleware := mw.JWTMiddleware()
	// Asumsi RBAC roles: report:admin untuk statistik global, report:read untuk detail student
	rbacAdmin := mw.RBACMiddleware("report:admin", authSvc) 
	rbacRead := mw.RBACMiddleware("report:read", authSvc)   

	reports := api.Group("/reports", jwtMiddleware)

	// GET /api/v1/reports/statistics (Hanya untuk Admin)
	reports.Get("/statistics", rbacAdmin, func(c *fiber.Ctx) error {
		// Ambil data User ID dan Role dari c.Locals (diset oleh JWTMiddleware)
        
        // --- PENGAMBILAN USER ID YANG ROBUST ---
        var userIDStr string
        userLocals := c.Locals("userID")
        
        if str, ok := userLocals.(string); ok {
            userIDStr = str
        } else if uid, ok := userLocals.(uuid.UUID); ok {
            userIDStr = uid.String()
        }

		// Ambil Role (Role harusnya selalu string)
		userRole, ok := c.Locals("role").(string)
		if !ok || userRole == "" || userIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID or role missing from token"})
		}
        
        // Parse ID ke UUID untuk service layer
        parsedUserID, err := uuid.Parse(userIDStr) 
        if err != nil {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
        }

		// Panggil service untuk mendapatkan statistik global
		result, err := achievementSvc.GetAchievementStatistics(c.Context(), userRole, parsedUserID)
		if err != nil {
			status := http.StatusBadRequest
			if err.Error() == "forbidden: hanya Admin yang dapat mengakses statistik global" {
				status = http.StatusForbidden
			}
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

    // GET /api/v1/reports/student/:id (Dapat diakses oleh Admin, Dosen Wali, Mahasiswa)
	reports.Get("/student/:id", rbacRead, func(c *fiber.Ctx) error {
		targetStudentID := c.Params("id")
		if targetStudentID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Student ID is required"})
		}
        
        // --- PENGAMBILAN USER ID YANG ROBUST ---
        var userIDStr string
        userLocals := c.Locals("userID")
        
        if str, ok := userLocals.(string); ok {
            userIDStr = str
        } else if uid, ok := userLocals.(uuid.UUID); ok {
            userIDStr = uid.String()
        }
        
        // Final check
        if userIDStr == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID missing or invalid type from token"})
        }
        
        parsedUserID, err := uuid.Parse(userIDStr) // Parse ke UUID untuk service
        if err != nil {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
        }
        
        // Ambil Role
        userRole, ok := c.Locals("role").(string)
        if !ok || userRole == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User role missing"})
        }
        // -------------------------------------------------------------

		// Panggil service yang sudah ada: ListAchievementsByStudentID
		result, err := achievementSvc.ListAchievementsByStudentID(c.Context(), targetStudentID, parsedUserID, userRole)
		if err != nil {
			status := http.StatusBadRequest
            if err.Error() == "forbidden: tidak memiliki hak akses untuk melihat prestasi mahasiswa ini" {
                status = http.StatusForbidden
            }
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})
}