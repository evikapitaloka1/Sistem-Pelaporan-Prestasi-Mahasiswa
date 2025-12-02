package routes

import (
    "net/http"

    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    mongo "uas/app/service/mongo"
    authService "uas/app/service/postgres"
    studentService "uas/app/service/postgres"
    mw "uas/middleware"
)

// StudentRoutes setup semua route terkait student
func StudentRoutes(
    api fiber.Router,
    authSvc *authService.AuthService,
    studentSvc studentService.StudentService,
    achievementSvc mongo.AchievementService,
) {
    // Asumsi: JWTMiddleware menyimpan token/claims di c.Locals
    jwtMiddleware := mw.JWTMiddleware()
    rbacRead := mw.RBACMiddleware("student:read", authSvc)
    rbacUpdate := mw.RBACMiddleware("student:update", authSvc)

    // Grup Route menggunakan jwtMiddleware secara default
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

        body := struct {
            NewAdvisorID string `json:"new_advisor_id"`
        }{}

        if err := c.BodyParser(&body); err != nil {
            return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
        }

        role := c.Locals("role").(string)

        err := studentSvc.UpdateAdvisor(c.Context(), studentID, body.NewAdvisorID, role)
        if err != nil {
            return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
        }

        return c.JSON(fiber.Map{"status": "success"})
    })

    // GET achievements by student ID (Perbaikan)
    students.Get("/:id/achievements", rbacRead, func(c *fiber.Ctx) error {
        // Ambil student ID dari path param
        targetStudentID := c.Params("id")
        if targetStudentID == "" {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Student ID is required",
            })
        }

        // --- 🛑 PERBAIKAN: PENGAMBILAN DAN VALIDASI USER ID DARI TOKEN LOCALS 🛑 ---
        var userIDStr string

        // 1️⃣ Ambil User ID dari locals (setelah middleware auth)
        userLocals := c.Locals("userID")
        if userLocals == nil {
             return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "User ID missing from token locals",
            })
        }
        
        // Coba konversi ke string
        if str, ok := userLocals.(string); ok {
            userIDStr = str
        } else if uid, ok := userLocals.(uuid.UUID); ok {
            // Jika disimpan sebagai objek UUID, konversi ke string
            userIDStr = uid.String()
        }

        if userIDStr == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "User ID missing or invalid type from token",
            })
        }
        // --------------------------------------------------------------------------

        // 2️⃣ Parse string ke UUID
        parsedUserID, err := uuid.Parse(userIDStr)
        if err != nil {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Invalid user ID format in token",
            })
        }

        // 3️⃣ Ambil Role dari locals
        userRole, ok := c.Locals("role").(string)
        if !ok || userRole == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "User role missing or invalid",
            })
        }

        // 4️⃣ Panggil service untuk ambil achievements
        result, err := achievementSvc.ListAchievementsByStudentID(c.Context(), targetStudentID, parsedUserID, userRole)
        if err != nil {
            // Menggunakan StatusUnauthorized jika errornya adalah forbidden/akses ditolak
            if err.Error() == "forbidden: tidak memiliki hak akses untuk melihat prestasi mahasiswa ini" {
                 return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                    "error": err.Error(),
                })
            }
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": err.Error(),
            })
        }

        // 5️⃣ Return hasil
        return c.JSON(fiber.Map{
            "status": "success",
            "data": result,
        })
    })
}