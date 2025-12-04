package routes

import (
	"net/http"
	"strings"
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
	jwtMiddleware fiber.Handler, // Menerima JWT Handler yang sudah siap (Blacklist Checker terpasang)
) {
	
	// Buat RBAC Middleware (yang membutuhkan authSvc, yang sekarang mengimplementasikan IAuthService yang diselaraskan)
	// Baris ini akan berhasil jika IAuthService sudah memiliki Logout(ctx, jti string) error
	rbacAdmin := mw.RBACMiddleware("report:admin", authSvc) 
	
	// Gunakan jwtMiddleware yang di-inject
	reports := api.Group("/reports", jwtMiddleware)

	// GET /api/v1/reports/statistics (Hanya untuk Admin)
	reports.Get("/statistics", rbacAdmin, func(c *fiber.Ctx) error {
		
		var userIDStr string
		userLocals := c.Locals("userID")
		
		if str, ok := userLocals.(string); ok {
			userIDStr = str
		} else if uid, ok := userLocals.(uuid.UUID); ok {
			userIDStr = uid.String()
		}

		userRole, ok := c.Locals("role").(string)
		if !ok || userRole == "" || userIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID or role missing from token"})
		}
		
		parsedUserID, err := uuid.Parse(userIDStr) 
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
		}

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
	reports.Get("/student/:id", func(c *fiber.Ctx) error { 
        targetStudentID := c.Params("id")
        if targetStudentID == "" {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Student ID is required"})
        }

        // --- PENGAMBILAN USER ID (tetap robust) ---
        var userIDStr string
        userLocals := c.Locals("userID")
        
        if str, ok := userLocals.(string); ok {
            userIDStr = str
        } else if uid, ok := userLocals.(uuid.UUID); ok {
            userIDStr = uid.String()
        }
        
        if userIDStr == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID missing or invalid type from token"})
        }
        
        parsedUserID, err := uuid.Parse(userIDStr)
        if err != nil {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
        }
        
        // Ambil Role
        userRole, ok := c.Locals("role").(string)
        if !ok || userRole == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User role missing"})
        }
        // -------------------------------------------------------------

        // 🛑 LAKUKAN LOGIKA SELF-ACCESS DI SINI 🛑
        isSelfAccess := targetStudentID == userIDStr
        
        // Jika bukan akses diri sendiri DAN user adalah Mahasiswa, tolak segera.
        if !isSelfAccess && userRole == "Mahasiswa" {
            return c.Status(http.StatusForbidden).JSON(fiber.Map{
                "code": http.StatusForbidden,
                "error": "Akses ditolak: Anda hanya dapat melihat laporan profil Anda sendiri.",
            })
        }
        // -------------------------------------------
        
        // 🚨 Panggil service laporan yang sesuai 🚨
        // Asumsi service Anda untuk laporan adalah reportSvc
        // GANTI INI:
        // result, err := achievementSvc.ListAchievementsByStudentID(c.Context(), targetStudentID, parsedUserID, userRole)
        
        // DENGAN FUNGSI REPORT:
        // result, err := reportSvc.GenerateStudentReport(c.Context(), targetStudentID, parsedUserID, userRole) // <-- Sesuaikan nama service
        
        
        // UNTUK SEMENTARA, SAYA ASUMSIKAN ANDA MEMANG INGIN MEMANGGIL ACHIEVEMENT SERVICE (HANYA UNTUK TUJUAN DEMO)
        result, err := achievementSvc.ListAchievementsByStudentID(c.Context(), targetStudentID, parsedUserID, userRole)


        if err != nil {
            status := http.StatusBadRequest
            // Jika service mengembalikan error otorisasi, pastikan kita mengembalikan 403
            if strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "tidak memiliki hak akses") {
                status = http.StatusForbidden
            }
            return c.Status(status).JSON(fiber.Map{"error": err.Error()})
        }
        return c.JSON(fiber.Map{"status": "success", "data": result})
    })
}