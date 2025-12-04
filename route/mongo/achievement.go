package routes

import (
	"context"
	"net/http"
	"fmt"   
    "time"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	// Service Interface untuk RBAC & Achievement
	achievementService "uas/app/service/mongo"
	authService "uas/app/service/postgres"

	// Model untuk request
	models "uas/app/model/mongo"

	// Middleware
	mw "uas/middleware"
)

// --- STRUCT LOKAL PENGGANTI yang Dihapus/Disederhanakan ---
// AchievementUpdateRequest (Dihapus, diganti dengan models.AchievementRequest)

// VerificationRequest dibuat sederhana untuk Verifikasi (Hanya jika models.go tidak punya)
// Catatan: Jika models.RejectRequest hanya berisi RejectionNote, maka VerificationRequest tidak dibutuhkan.

// Definisikan tipe yang akan mengimplementasikan Blacklist Checker
type NoopBlacklistChecker struct{}

func (n *NoopBlacklistChecker) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

// AchievementRoutes sekarang menerima 2 Service Instance
func AchievementRoutes(
	api fiber.Router,
	authSvc *authService.AuthService,
	achievementSvc achievementService.AchievementService,
) {
	// --- Inisialisasi Middleware ---
	jwtMiddleware := mw.JWTMiddleware(&NoopBlacklistChecker{})

	// Middleware RBAC untuk setiap Permission
	rbacCreate := mw.RBACMiddleware("achievement:create", authSvc)
	rbacUpdate := mw.RBACMiddleware("achievement:update", authSvc)
	rbacDelete := mw.RBACMiddleware("achievement:delete", authSvc)
	rbacVerify := mw.RBACMiddleware("achievement:verify", authSvc)
	rbacView := mw.RBACMiddleware("achievement:read", authSvc)

	// Group route /achievements dengan JWT
	achievements := api.Group("/achievements", jwtMiddleware)

	// --- FUNGSI HELPER YANG DIPERBAIKI (Konversi string ke uuid.UUID) ---
	getUserData := func(c *fiber.Ctx) (uuid.UUID, string, error) {
		userIDVal := c.Locals("userID")

		// 1. Ambil nilai 'userID' sebagai STRING dari c.Locals
		userIDStr, ok := userIDVal.(string)
		if !ok || userIDStr == "" {
			return uuid.Nil, "", fiber.NewError(http.StatusUnauthorized, "User ID not found in context or invalid type (expected string)")
		}

		// 2. Konversi STRING ke objek uuid.UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			// Menangani kasus string yang ada, tetapi bukan format UUID yang valid
			return uuid.Nil, "", fiber.NewError(http.StatusUnauthorized, "User ID string in context is not a valid UUID format")
		}

		userRoleVal := c.Locals("role")
		userRole, okRole := userRoleVal.(string)
		if !okRole || userRole == "" {
			return uuid.Nil, "", fiber.NewError(http.StatusUnauthorized, "Role not found in context or invalid type")
		}
		return userID, userRole, nil
	}

	// ------------------------------------------
	// 1. ENDPOINT UMUM (READ/HISTORY)
	// ------------------------------------------

	// GET /achievements - List Achievement (Filtered berdasarkan role)
	achievements.Get("/", rbacView, func(c *fiber.Ctx) error {
		userID, userRole, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		result, serviceErr := achievementSvc.ListAchievements(c.Context(), userRole, userID)
		if serviceErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": serviceErr.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	// GET /achievements/:id - Detail Achievement
	achievements.Get("/:id", rbacView, func(c *fiber.Ctx) error {
		achievementID := c.Params("id")
		userID, userRole, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		result, serviceErr := achievementSvc.GetAchievementDetail(c.Context(), achievementID, userID, userRole)
		if serviceErr != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": serviceErr.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	// GET /achievements/:id/history - Riwayat Verifikasi
	achievements.Get("/:id/history", rbacView, func(c *fiber.Ctx) error {
		achievementID := c.Params("id")
		userID, userRole, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		result, serviceErr := achievementSvc.GetAchievementHistory(c.Context(), achievementID, userID, userRole)
		if serviceErr != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": serviceErr.Error()})
		}

		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	// ------------------------------------------
	// 2. MAHASISWA (CREATE, UPDATE, DELETE, SUBMIT)
	// ------------------------------------------

	// POST /achievements - Buat Achievement Baru
	achievements.Post("/", rbacCreate, func(c *fiber.Ctx) error {
		var req models.AchievementRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body", "details": err.Error()})
		}

		userID, userRole, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		result, serviceErr := achievementSvc.CreateAchievement(c.Context(), userID, userRole, req)

		if serviceErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": serviceErr.Error()})
		}

		return c.Status(http.StatusCreated).JSON(fiber.Map{"status": "success", "data": result})
	})

	// PUT /achievements/:id - Update Achievement
	achievements.Put("/:id", rbacUpdate, func(c *fiber.Ctx) error {
		achievementID := c.Params("id")
		// MENGGUNAKAN struct DARI models.AchievementRequest
		var req models.AchievementRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body", "details": err.Error()})
		}

		userID, userRole, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		// Mengasumsikan service.UpdateAchievement menerima models.AchievementRequest
		// Tidak perlu mapping karena langsung menggunakan models.AchievementRequest
		serviceErr := achievementSvc.UpdateAchievement(c.Context(), achievementID, userID, userRole, req)

		if serviceErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": serviceErr.Error()})
		}

		return c.JSON(fiber.Map{"status": "success", "message": "Achievement updated successfully"})
	})

	// DELETE /achievements/:id - Hapus Achievement
	achievements.Delete("/:id", rbacDelete, func(c *fiber.Ctx) error {
		achievementID := c.Params("id")

		userID, userRole, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		serviceErr := achievementSvc.DeleteAchievement(c.Context(), achievementID, userID, userRole)
		if serviceErr != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": serviceErr.Error()})
		}

		return c.SendStatus(http.StatusNoContent)
	})

	// POST /achievements/:id/submit - Mengajukan Achievement untuk Verifikasi
	achievements.Post("/:id/submit", rbacUpdate, func(c *fiber.Ctx) error {
		achievementID := c.Params("id")

		userID, _, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		serviceErr := achievementSvc.SubmitForVerification(c.Context(), achievementID, userID)

		if serviceErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": serviceErr.Error()})
		}

		return c.JSON(fiber.Map{"status": "success", "message": "Achievement submitted for verification"})
	})


	// ------------------------------------------
	// 3. VERIFIKATOR (VERIFY, REJECT)
	// ------------------------------------------

	// PUT /achievements/:id/verify - Verifikasi Achievement
	achievements.Put("/:id/verify", rbacVerify, func(c *fiber.Ctx) error {
		achievementID := c.Params("id")
		// Catatan: Jika service.VerifyAchievement tidak membutuhkan body, 
		// tidak perlu ada body parser dan struct request lokal.

		userID, _, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		// Want: (context.Context, string, "github.com/google/uuid".UUID)
		serviceErr := achievementSvc.VerifyAchievement(c.Context(), achievementID, userID)

		if serviceErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": serviceErr.Error()})
		}

		return c.JSON(fiber.Map{"status": "success", "message": "Achievement verified successfully"})
	})

	// PUT /achievements/:id/reject - Tolak Achievement
	achievements.Put("/:id/reject", rbacVerify, func(c *fiber.Ctx) error {
		achievementID := c.Params("id")
		// Menggunakan models.RejectRequest dari package models
		var req models.RejectRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.RejectionNote == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Rejection reason (rejection_note) is required"})
		}

		// userID SUDAH bertipe uuid.UUID karena perubahan di getUserData
		userID, _, err := getUserData(c)
		if err != nil {
			return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
		}

		// Memanggil service.RejectAchievement dengan 4 argumen: (ctx, id, userID, RejectionNote)
		serviceErr := achievementSvc.RejectAchievement(c.Context(), achievementID, userID, req.RejectionNote)

		if serviceErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": serviceErr.Error()})
		}

		return c.JSON(fiber.Map{"status": "success", "message": "Achievement rejected successfully"})
	})
	achievements.Post("/:id/attachments", func(c *fiber.Ctx) error {
    achievementID := c.Params("id")

    // Ambil userID dan role dari JWT
    userIDRaw := c.Locals("userID")
    if userIDRaw == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "User ID missing from token",
        })
    }

    userID, err := uuid.Parse(fmt.Sprint(userIDRaw))
    if err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid userID from token",
        })
    }

    // Ambil file dari form-data
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "file wajib diupload",
        })
    }

    // Baca file
    fileSrc, err := file.Open()
    if err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "gagal membuka file",
        })
    }
    defer fileSrc.Close()

    // Baca bytes
    fileUrl := fmt.Sprintf("/uploads/achievements/%s/%s", achievementID, file.Filename)

    // Buat struct Attachment
     attachment := models.Attachment{
        FileName:   file.Filename,
        FileType:   file.Header.Get("Content-Type"),
        FileUrl:    fileUrl,
        UploadedAt: time.Now(),
    }

    // Panggil service
    err = achievementSvc.AddAttachment(c.Context(), achievementID, userID, attachment)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "status": "success",
        "message": "attachment berhasil ditambahkan",
    })
})


}
