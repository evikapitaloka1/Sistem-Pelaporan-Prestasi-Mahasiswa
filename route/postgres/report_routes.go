package routes

import (
	"github.com/gofiber/fiber/v2"
	"strings"

	mongo "uas/app/service/mongo"
	authService "uas/app/service/postgres"

	"uas/helper"
)

func ReportRoutes(
	api fiber.Router,
	authSvc *authService.AuthService,
	achievementSvc mongo.AchievementService,
	jwtMiddleware fiber.Handler,
) {

	reports := api.Group("/reports", jwtMiddleware)

	// =====================================================
	// GET /statistics (Admin only)
	// =====================================================
	reports.Get("/statistics", func(c *fiber.Ctx) error {

		userID, err := helper.GetUserID(c)
		if err != nil {
			return helper.SendError(c, fiber.StatusUnauthorized, err.Error())
		}

		role, err := helper.GetRole(c)
		if err != nil {
			return helper.SendError(c, fiber.StatusUnauthorized, err.Error())
		}

		result, err := achievementSvc.GetAchievementStatistics(c.Context(), role, userID)
		if err != nil {

			if strings.Contains(err.Error(), "forbidden") {
				return helper.SendError(c, fiber.StatusForbidden, err.Error())
			}

			return helper.SendError(c, fiber.StatusInternalServerError, err.Error())
		}

		return helper.SendSuccess(c, result)
	})

	// =====================================================
	// GET /student/:id (Admin, Dosen, Mahasiswa-self)
	// =====================================================
	reports.Get("/student/:id", func(c *fiber.Ctx) error {

		targetID := c.Params("id")
		if targetID == "" {
			return helper.SendError(c, fiber.StatusBadRequest, "student ID is required")
		}

		// Ambil user dari token
		userID, err := helper.GetUserID(c)
		if err != nil {
			return helper.SendError(c, fiber.StatusUnauthorized, err.Error())
		}

		role, err := helper.GetRole(c)
		if err != nil {
			return helper.SendError(c, fiber.StatusUnauthorized, err.Error())
		}

		// Mahasiswa hanya boleh akses dirinya sendiri
		isSelf := targetID == userID.String()
		if role == "Mahasiswa" && !isSelf {
			return helper.SendError(c, fiber.StatusForbidden,
				"akses ditolak: mahasiswa hanya dapat melihat laporan dirinya sendiri")
		}

		result, err := achievementSvc.ListAchievementsByStudentID(
			c.Context(),
			targetID,
			userID,
			role,
		)

		if err != nil {
			if strings.Contains(err.Error(), "forbidden") {
				return helper.SendError(c, fiber.StatusForbidden, err.Error())
			}
			return helper.SendError(c, fiber.StatusBadRequest, err.Error())
		}

		return helper.SendSuccess(c, result)
	})
}
