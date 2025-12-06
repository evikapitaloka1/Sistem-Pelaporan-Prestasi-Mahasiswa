package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"uas/helper"
	mw "uas/middleware"

	achievementService "uas/app/service/mongo"
	authService "uas/app/service/postgres"
	postgres "uas/app/service/postgres"
	models "uas/app/model/mongo"
)

func AchievementRoutes(
	api fiber.Router,
	authSvc *authService.AuthService,
	achievementSvc achievementService.AchievementService,
	studentSvc postgres.StudentService,
) {
	jwtMW := mw.JWTMiddleware(&helper.NoopBlacklistChecker{})

	rbacCreate := mw.RBACMiddleware("achievement:create", authSvc)
	rbacUpdate := mw.RBACMiddleware("achievement:update", authSvc)
	rbacDelete := mw.RBACMiddleware("achievement:delete", authSvc)
	rbacVerify := mw.RBACMiddleware("achievement:verify", authSvc)
	rbacView := mw.RBACMiddleware("achievement:read", authSvc)

	ach := api.Group("/achievements", jwtMW)

	// LIST =================================================
	ach.Get("/", rbacView, func(c *fiber.Ctx) error {
		userID, role, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		data, e := achievementSvc.ListAchievements(c.Context(), role, userID)
		if e != nil {
			return helper.SendError(c, fiber.StatusInternalServerError, e.Error())
		}

		return helper.SendSuccess(c, data)
	})

	// DETAIL ==============================================
	ach.Get("/:id", rbacView, func(c *fiber.Ctx) error {
		id := c.Params("id")
		userID, role, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		data, e := achievementSvc.GetAchievementDetail(c.Context(), id, userID, role)
		if e != nil {
			return helper.SendError(c, fiber.StatusNotFound, e.Error())
		}

		return helper.SendSuccess(c, data)
	})

	// HISTORY =============================================
	ach.Get("/:id/history", rbacView, func(c *fiber.Ctx) error {
		id := c.Params("id")
		userID, role, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		data, e := achievementSvc.GetAchievementHistory(c.Context(), id, userID, role)
		if e != nil {
			return helper.SendError(c, fiber.StatusNotFound, e.Error())
		}

		return helper.SendSuccess(c, data)
	})

	// CREATE ==============================================
	ach.Post("/", rbacCreate, func(c *fiber.Ctx) error {
		var req models.AchievementRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.SendError(c, fiber.StatusBadRequest, "invalid body")
		}

		userID, role, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		res, e := achievementSvc.CreateAchievement(c.Context(), userID, role, req)
		if e != nil {
			return helper.SendError(c, fiber.StatusBadRequest, e.Error())
		}

		return helper.Created(c, res)
	})

	// UPDATE ==============================================
	ach.Put("/:id", rbacUpdate, func(c *fiber.Ctx) error {
		id := c.Params("id")
		var req models.AchievementRequest

		if err := c.BodyParser(&req); err != nil {
			return helper.SendError(c, fiber.StatusBadRequest, "invalid body")
		}

		userID, role, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		if e := achievementSvc.UpdateAchievement(c.Context(), id, userID, role, req); e != nil {
			return helper.SendError(c, fiber.StatusBadRequest, e.Error())
		}

		return helper.SendSuccessNoData(c)
	})

	// DELETE ==============================================
	ach.Delete("/:id", rbacDelete, func(c *fiber.Ctx) error {
		id := c.Params("id")
		userID, role, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		if e := achievementSvc.DeleteAchievement(c.Context(), id, userID, role); e != nil {
			return helper.SendError(c, fiber.StatusNotFound, e.Error())
		}

		return c.SendStatus(http.StatusNoContent)
	})

	// SUBMIT ==============================================
	ach.Post("/:id/submit", rbacUpdate, func(c *fiber.Ctx) error {
		id := c.Params("id")
		userID, _, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		if e := achievementSvc.SubmitForVerification(c.Context(), id, userID); e != nil {
			return helper.SendError(c, fiber.StatusBadRequest, e.Error())
		}

		return helper.SendSuccessNoData(c)
	})

	// VERIFY ==============================================
	ach.Post("/:id/verify", rbacVerify, func(c *fiber.Ctx) error {
		id := c.Params("id")
		userID, _, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		if e := achievementSvc.VerifyAchievement(c.Context(), id, userID); e != nil {
			return helper.SendError(c, fiber.StatusBadRequest, e.Error())
		}

		return helper.SendSuccessNoData(c)
	})

	// REJECT ==============================================
	ach.Post("/:id/reject", rbacVerify, func(c *fiber.Ctx) error {
		id := c.Params("id")

		var req models.RejectRequest
		if err := c.BodyParser(&req); err != nil || req.RejectionNote == "" {
			return helper.SendError(c, fiber.StatusBadRequest, "Rejection note required")
		}

		userID, _, err := helper.GetUserData(c)
		if err != nil {
			return FiberError(c, err)
		}

		if e := achievementSvc.RejectAchievement(c.Context(), id, userID, req.RejectionNote); e != nil {
			return helper.SendError(c, fiber.StatusBadRequest, e.Error())
		}

		return helper.SendSuccessNoData(c)
	})

	// ATTACHMENT ==========================================
	ach.Post("/:id/attachments", func(c *fiber.Ctx) error {
    mongoID := c.Params("id")

    // Ambil userID dari JWT
    raw := c.Locals("userID")
    userID, err := uuid.Parse(fmt.Sprint(raw))
    if err != nil {
        return helper.SendError(c, fiber.StatusUnauthorized, "invalid user id")
    }

    // Ambil role dari JWT
    roleRaw := c.Locals("role")
    role := ""
    if roleRaw != nil {
        role = fmt.Sprint(roleRaw)
    }

    // Hanya mahasiswa dan admin yang boleh upload
    if role != "mahasiswa" && role != "admin" {
        return helper.SendError(c, fiber.StatusForbidden, "akses ditolak: hanya mahasiswa atau admin yang bisa upload")
    }

    // Panggil service untuk validasi, tapi hasilnya tidak perlu disimpan
    if _, err := achievementSvc.GetAchievementDetail(c.Context(), mongoID, userID, role); err != nil {
        return helper.SendError(c, fiber.StatusForbidden, err.Error())
    }

    // Ambil file dari request
    file, err := c.FormFile("file")
    if err != nil {
        return helper.SendError(c, fiber.StatusBadRequest, "file required")
    }

    // Buat attachment
    attachment := models.Attachment{
        FileName:   file.Filename,
        FileType:   file.Header.Get("Content-Type"),
        FileUrl:    fmt.Sprintf("/uploads/achievements/%s/%s", mongoID, file.Filename),
        UploadedAt: time.Now(),
    }

    // Simpan attachment via service
    if err := achievementSvc.AddAttachment(c.Context(), mongoID, userID, attachment); err != nil {
        return helper.SendError(c, fiber.StatusBadRequest, err.Error())
    }

    return helper.SendSuccess(c, attachment)
})

}

// Wrapper untuk fiber.Error
func FiberError(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		return helper.SendError(c, fe.Code, fe.Message)
	}
	return helper.SendError(c, fiber.StatusInternalServerError, err.Error())
}
