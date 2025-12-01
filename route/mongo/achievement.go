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

	// Di dalam AchievementRoutes (uas/routes/achievement.go)
// ----------------------
// A. ENDPOINT UMUM (READ/HISTORY)
// ----------------------
// ... (Kode ListAchievements dan GetAchievementDetail)

achievements.Get("/:id/history", rbacView, func(c *fiber.Ctx) error {
    // 1. Ambil ID Prestasi (MongoDB ObjectID) dari parameter
    achievementID := c.Params("id")
    if len(achievementID) != 24 {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid achievement ID format (Expected 24-char ObjectID)"})
    }

    // 2. Ambil User ID dari context (diperlukan untuk otorisasi di service)
    userID, err := getUserID(c)
    if err != nil {
        return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
    }
    
    // 3. Ambil Role dari context (diperlukan untuk otorisasi di service)
    role, err := getRole(c)
    if err != nil {
        return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
    }

    // 4. Panggil Service Layer untuk mengambil riwayat status
    // ASUMSI: GetAchievementHistory sudah ditambahkan ke AchievementService interface Anda
    result, err := achievementSvc.GetAchievementHistory(c.Context(), achievementID, userID, role)
    
    if err != nil {
        // Asumsi status 404 jika tidak ditemukan, atau 400/500 untuk error lain
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
    }

    // 5. Kembalikan data riwayat dalam format JSON
    return c.JSON(fiber.Map{"status": "success", "data": result})
})

// ... (Kode endpoint lainnya)

	// ----------------------
	// B. MAHASISWA (CREATE, UPDATE, DELETE)
	// ----------------------
achievements.Post("/", rbacCreate, func(c *fiber.Ctx) error {
    var req models.AchievementRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body", "details": err.Error()})
    }
    
    // 1. Ambil User ID
    userID, err := getUserID(c)
    if err != nil {
        return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
    }
    
    // 2. 🛑 TAMBAH: Ambil User Role dari context
    userRole, errRole := getRole(c)
    if errRole != nil {
        // Jika role hilang atau tidak valid, tolak akses
        return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": errRole.Error()})
    }
    
    // 3. KOREKSI PANGGILAN SERVICE: Tambahkan userRole
    result, err := achievementSvc.CreateAchievement(c.Context(), userID, userRole, req)
    
    if err != nil {
        // Gunakan 400 Bad Request jika error berasal dari logic bisnis/validasi
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }
    
    return c.Status(http.StatusCreated).JSON(fiber.Map{"status": "success", "data": result})
})

	// File: uas/route/mongo/achievement.go

achievements.Put("/:id", rbacUpdate, func(c *fiber.Ctx) error {
    // 1. Ambil ID MongoDB dari parameter
    achievementID := c.Params("id")
    if len(achievementID) != 24 {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid achievement ID format (Expected 24-char ObjectID)"})
    }
    
    // 2. Ambil User ID dan Role dari context
    userID, err := getUserID(c)
    if err != nil {
        return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
    }
    
    userRole, err := getRole(c) // Asumsi getRole sudah ada di file ini
    if err != nil {
        return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
    }

    // 3. Parsing Request Body
    var req models.AchievementRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body", "details": err.Error()})
    }

    // 4. Panggil Service Layer
    // NOTE: Pastikan AchievementService interface Anda sudah diupdate!
    err = achievementSvc.UpdateAchievement(c.Context(), achievementID, userID, userRole, req)
    
    if err != nil {
        // Tangani error spesifik dari service (misal: "forbidden", "not found", "status draft required")
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }

    // 5. Sukses
    return c.JSON(fiber.Map{"status": "success", "message": "Achievement updated successfully"})
})
// Perbaikan pada handler achievements.Delete("/:id", rbacDelete, ...)

achievements.Delete("/:id", rbacDelete, func(c *fiber.Ctx) error {
    // 1. Ambil ID MongoDB
    achievementID := c.Params("id")
    if len(achievementID) != 24 {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid achievement ID format (Expected 24-char ObjectID)"})
    }

    // 2. Ambil User ID
    userID, err := getUserID(c)
    if err != nil {
        return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
    }
    
    // 3. Ambil User Role
    // 🛑 PERBAIKAN ERROR 1: Gunakan operator assignment pendek 'a, err := b()' hanya jika 'err' adalah variabel baru.
    // Jika 'err' sudah ada di scope, gunakan '=' untuk assignment.
    userRole, errRole := getRole(c) // <-- Gunakan variabel baru (errRole) untuk menghindari duplikasi Baris 175
    if errRole != nil {
        // Jika getRole gagal (misal: role hilang), kembalikan error
        return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": errRole.Error()})
    }

    // 4. Panggil Service Layer (Soft Delete)
    // Gunakan 'err' dari scope luar
    err = achievementSvc.DeleteAchievement(c.Context(), achievementID, userID, userRole) 
    
    if err != nil {
        // Tangani error jika service gagal
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }

    // 5. Sukses (WAJIB ADA)
    // 🛑 PERBAIKAN ERROR 2: Pastikan ada return statement di akhir fungsi (Baris 183)
    return c.JSON(fiber.Map{"status": "success", "message": "Achievement deleted (soft delete)"})
}) // Fungsi berakhir di sini
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
	// Di routes/achievement.go, di bawah bagian B. MAHASISWA

// Route baru untuk Submit for verification (Mahasiswa)
achievements.Post("/:id/submit", func(c *fiber.Ctx) error {
    achievementID := c.Params("id")
    if len(achievementID) != 24 {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid achievement ID format"})
    }
    
    // Ambil User ID (Mahasiswa)
    userID, err := getUserID(c)
    if err != nil { return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()}) }

    // Panggil Service SubmitForVerification
    err = achievementSvc.SubmitForVerification(c.Context(), achievementID, userID)
    
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }
    return c.JSON(fiber.Map{"status": "success", "message": "Achievement submitted for verification"})
})
}
