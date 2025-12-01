package routes

import (
	"net/http"
	"github.com/gofiber/fiber/v2"
	authService "uas/app/service/postgres"
	studentService "uas/app/service/postgres" // Asumsi path service
	mw "uas/middleware" // Asumsi path middleware
)

// DTO untuk body request PUT /advisor
type UpdateAdvisorRequest struct {
    NewAdvisorID string `json:"advisorId"` // Menggunakan nama field yang konsisten
}

// StudentRoutes mendefinisikan rute untuk Student.
func StudentRoutes(
	api fiber.Router,
	authSvc *authService.AuthService, // Asumsi Auth Service ada untuk RBAC
	studentSvc studentService.StudentService,
) {
	jwtMiddleware := mw.JWTMiddleware()
	rbacRead := mw.RBACMiddleware("student:read", authSvc)
	rbacUpdate := mw.RBACMiddleware("student:update", authSvc)

	students := api.Group("/students", jwtMiddleware)

	// GET /api/v1/students (List semua students)
	students.Get("/", rbacRead, func(c *fiber.Ctx) error {
		result, err := studentSvc.ListStudents(c.Context())
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	// GET /api/v1/students/:id (Detail student)
	students.Get("/:id", rbacRead, func(c *fiber.Ctx) error {
		studentID := c.Params("id")
		result, err := studentSvc.GetStudentDetail(c.Context(), studentID)
		if err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
	})

	// PUT /api/v1/students/:id/advisor (Update Dosen Wali)
	students.Put("/:id/advisor", rbacUpdate, func(c *fiber.Ctx) error {
		studentID := c.Params("id")
		var req UpdateAdvisorRequest
        
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		// Ambil Role dari context (asumsi getRole ada atau sudah di setup)
        callerRole, _ := c.Locals("role").(string)

		err := studentSvc.UpdateAdvisor(c.Context(), studentID, req.NewAdvisorID, callerRole)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "message": "Dosen wali berhasil diupdate"})
	})
}