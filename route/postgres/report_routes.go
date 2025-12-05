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
    
    // Gunakan fiber.Ctx untuk Context yang memiliki nilai timeout.
    ctx := c.Context() 
    
    // --- 1. Ekstraksi dan Validasi User ID ---
    
    // Karena rbacAdmin sudah dijalankan, asumsi userID dan role sudah ada 
    // dan role adalah 'admin'. Kita prioritaskan tipe data yang konsisten.
    userIDStr, ok := c.Locals("userID").(string)
    userRole, roleOk := c.Locals("role").(string)

    // Jika middleware Anda menyimpan UUID sebagai uuid.UUID (lebih bersih),
    // gunakan: parsedUserID, ok := c.Locals("userID").(uuid.UUID)

    if !ok || !roleOk || userIDStr == "" || userRole == "" {
        // Status 401 adalah tepat jika data token hilang di Locals (safety net)
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID or role missing from token"})
    }

    parsedUserID, err := uuid.Parse(userIDStr) 
    if err != nil {
        // Status 401 juga tepat jika format ID di token tidak valid
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
    }

    // --- 2. Panggil Service untuk mendapatkan semua data statistik ---
    // Gunakan ctx yang diambil dari c.Context()
    result, err := achievementSvc.GetAchievementStatistics(ctx, userRole, parsedUserID)
    
    if err != nil {
        // Set default status ke 500 (Internal Server Error)
        status := fiber.StatusInternalServerError 
        errorMessage := err.Error()

        // Penanganan spesifik untuk error otorisasi (Forbidden)
        if strings.Contains(errorMessage, "forbidden") {
            status = fiber.StatusForbidden // 403
            errorMessage = "Akses ditolak. Anda tidak memiliki izin untuk melihat statistik ini."
        }
        
        // Penanganan jika ada error validasi di service (optional, umumnya 400)
        // if strings.Contains(errorMessage, "invalid input") { status = fiber.StatusBadRequest }

        return c.Status(status).JSON(fiber.Map{"error": errorMessage})
    }
    
    // --- 3. Mengembalikan Respons ---
    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "status": "success", 
        "data": result,
    })
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