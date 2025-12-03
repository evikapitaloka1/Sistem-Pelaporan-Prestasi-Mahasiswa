package routes

import (
	
	"net/http"
	"strings" // Digunakan untuk strings.Contains
	
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	
	authService "uas/app/service/postgres"
	lecturerService "uas/app/service/postgres"
	// mw "uas/middleware" // Dihapus karena otorisasi penuh di Service Layer
)

// SetupLecturerRoutes mendaftarkan semua endpoint yang berkaitan dengan Dosen.
func SetupLecturerRoutes(
	api fiber.Router,
	authSvc *authService.AuthService, // Digunakan jika ada middleware RBAC
	lecturerSvc lecturerService.LecturerService,
	jwtMiddleware fiber.Handler, // Middleware JWT wajib
) {
	
	// Group route /lecturers dengan JWT
	lecturers := api.Group("/lecturers", jwtMiddleware)
	
	// GET /api/v1/lecturers
	// Logika: Admin -> Semua Dosen; Dosen -> Hanya data diri sendiri.
	lecturers.Get("/", func(c *fiber.Ctx) error {
		
		// --- PENGAMBILAN ID PENGGUNA YANG LOGIN (Wajib untuk Service) ---
		var userIDStr string
		userLocals := c.Locals("userID")
		role, _ := c.Locals("role").(string) // Ambil Role
		
		if str, ok := userLocals.(string); ok {
			userIDStr = str
		} else if uid, ok := userLocals.(uuid.UUID); ok {
			userIDStr = uid.String()
		}

		if userIDStr == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "User ID missing or invalid type from token"})
		}
		
		// Parse ke UUID (Wajib untuk Service Layer)
		requestingUserID, err := uuid.Parse(userIDStr) 
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
		}
		
		// -------------------------------------------------------------------------
		
		// Panggil Service Layer dengan ID dan Role untuk filtering
		result, err := lecturerSvc.GetAllLecturers(c.Context(), requestingUserID, role)
		if err != nil {
			// Jika error adalah "unauthorized" dari Service Layer, kembalikan 403 Forbidden.
			if strings.Contains(err.Error(), "unauthorized") {
				return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
			}
			// Jika error lain (e.g., DB Error, data tidak ditemukan)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()}) // Diperbaiki ke 500
		}
		
		if len(result) == 0 {
			// Mengembalikan 200 OK dengan array kosong jika tidak ada data
			return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": []interface{}{}, "message": "No lecturers found"})
		}
		
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})
	
	// GET /api/v1/lecturers/:id/advisees
	// Logika: Dosen hanya lihat bimbingan dari ID diri sendiri.
	lecturers.Get("/:id/advisees", func(c *fiber.Ctx) error {
		// 1. Ambil ID Dosen (UUID) yang diminta dari parameter URL
		lecturerIDStr := c.Params("id")
		targetLecturerID, err := uuid.Parse(lecturerIDStr)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lecturer ID format (Expected UUID)"})
		}
		
		// 2. --- PENGAMBILAN ID PENGGUNA YANG LOGIN (Wajib untuk Service) ---
		var userIDStr string
		userLocals := c.Locals("userID")
		
		// Ambil User ID
		if str, ok := userLocals.(string); ok {
			userIDStr = str
		} else if uid, ok := userLocals.(uuid.UUID); ok {
			userIDStr = uid.String()
		}

		if userIDStr == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "User ID missing or invalid type from token"})
		}

		// Parse string ke UUID (Wajib untuk Service Layer)
		requestingUserID, err := uuid.Parse(userIDStr) 
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
		}
		
		// Ambil Role
		role, ok := c.Locals("role").(string)
		if !ok || role == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "user role missing from token"})
		}
		// ---------------------------------------------------------------------------
		
		// 3. Panggil Service Layer: Logika otorisasi penuh ada di dalam service.
		result, err := lecturerSvc.GetAdvisees(c.Context(), targetLecturerID, requestingUserID, role)
		
		if err != nil {
			// Jika error adalah "unauthorized" dari Service Layer (akses ke bimbingan dosen lain)
			if strings.Contains(err.Error(), "unauthorized") {
				return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
			}
			// Jika error adalah "not found" (contoh: dari service layer)
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no advisees found") {
				return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			// Jika error lain (e.g., DB Error)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()}) // Diperbaiki ke 500
		}

		// 4. Kembalikan data
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})
}