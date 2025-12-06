package routes

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strings"
	"uas/helper"
	authService "uas/app/service/postgres"
	lecturerService "uas/app/service/postgres"
)

func SetupLecturerRoutes(
	api fiber.Router,
	authSvc *authService.AuthService,
	lecturerSvc lecturerService.LecturerService,
	jwtMiddleware fiber.Handler,
) {

	lecturers := api.Group("/lecturers", jwtMiddleware)

	// ====================== GET ALL LECTURERS ======================
	lecturers.Get("/", func(c *fiber.Ctx) error {

		userID, err := helper.GetUserIDFromContext(c)
		if err != nil {
			return helper.JsonError(c, http.StatusUnauthorized, err.Error())
		}

		role, err := helper.GetRoleFromContext(c)
		if err != nil {
			return helper.JsonError(c, http.StatusUnauthorized, err.Error())
		}

		result, err := lecturerSvc.GetAllLecturers(c.Context(), userID, role)
		if err != nil {
			if helper.IsUnauthorizedErr(err) {
				return helper.JsonError(c, http.StatusForbidden, err.Error())
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		if len(result) == 0 {
			return c.JSON(fiber.Map{
				"status":  "success",
				"data":    []interface{}{},
				"message": "No lecturers found",
			})
		}

		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	// ====================== GET ADVISEES BY LECTURER ID ======================
	lecturers.Get("/:id/advisees", func(c *fiber.Ctx) error {

		lecturerID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "Invalid lecturer ID format")
		}

		userID, err := helper.GetUserIDFromContext(c)
		if err != nil {
			return helper.JsonError(c, http.StatusUnauthorized, err.Error())
		}

		role, err := helper.GetRoleFromContext(c)
		if err != nil {
			return helper.JsonError(c, http.StatusUnauthorized, err.Error())
		}

		result, err := lecturerSvc.GetAdvisees(c.Context(), lecturerID, userID, role)
		if err != nil {
			if helper.IsUnauthorizedErr(err) {
				return helper.JsonError(c, http.StatusForbidden, err.Error())
			}

			if strings.Contains(err.Error(), "not found") {
				return helper.JsonError(c, http.StatusNotFound, err.Error())
			}

			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		return c.JSON(fiber.Map{"status": "success", "data": result})
	})
}
