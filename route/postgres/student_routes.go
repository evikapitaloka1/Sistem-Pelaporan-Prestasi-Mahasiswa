package routes

import (
	"net/http"
	"github.com/gofiber/fiber/v2"
	authService "uas/app/service/postgres"
	studentService "uas/app/service/postgres"
	mw "uas/middleware"
)

func StudentRoutes(
	api fiber.Router,
	authSvc *authService.AuthService,
	studentSvc studentService.StudentService,
) {
	jwtMiddleware := mw.JWTMiddleware()
	rbacRead := mw.RBACMiddleware("student:read", authSvc)
	rbacUpdate := mw.RBACMiddleware("student:update", authSvc)

	students := api.Group("/students", jwtMiddleware)

	// GET list
	students.Get("/", rbacRead, func(c *fiber.Ctx) error {
		result, err := studentSvc.ListStudents(c.Context())
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	// GET detail
	students.Get("/:id", rbacRead, func(c *fiber.Ctx) error {
		studentID := c.Params("id")
		result, err := studentSvc.GetStudentDetail(c.Context(), studentID)
		if err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	// PUT update advisor
	students.Put("/:id/advisor", rbacUpdate, func(c *fiber.Ctx) error {
		studentID := c.Params("id")

		// Tanpa model
		body := struct {
			NewAdvisorID string `json:"new_advisor_id"`
		}{}

		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		role := c.Locals("role").(string)

		err := studentSvc.UpdateAdvisor(c.Context(), studentID, body.NewAdvisorID, role)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "success"})
	})
}
