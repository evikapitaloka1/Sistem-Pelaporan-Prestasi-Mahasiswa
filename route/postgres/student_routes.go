package routes

import (
    "github.com/gofiber/fiber/v2"
    mongo "uas/app/service/mongo"
    authService "uas/app/service/postgres"
    studentService "uas/app/service/postgres"
    mw "uas/middleware"
    "uas/helper"
)

func StudentRoutes(
    api fiber.Router,
    authSvc *authService.AuthService,
    studentSvc studentService.StudentService,
    achievementSvc mongo.AchievementService,
    jwtMiddleware fiber.Handler,
) {
    rbacUpdate := mw.RBACMiddleware("student:update", authSvc)
    students := api.Group("/students", jwtMiddleware)

    // -----------------------------
    // 1. GET List Students
    // -----------------------------
    students.Get("/", func(c *fiber.Ctx) error {
        ctx, userID, role, err := helper.ExtractUserContext(c)
        if err != nil {
            return helper.ErrorResponse(c, fiber.StatusUnauthorized, err)
        }

        result, err := studentSvc.ListStudents(ctx, userID, role)
        return helper.ServiceResponse(c, result, err)
    })

    // -----------------------------
    // 2. GET Detail Student
    // -----------------------------
    students.Get("/:id", func(c *fiber.Ctx) error {

    ctx, userID, role, err := helper.ExtractUserContext(c)
    if err != nil {
        return helper.ErrorResponse(c, fiber.StatusUnauthorized, err)
    }

    // Ambil detail student dulu
    student, err := studentSvc.GetStudentDetail(ctx, c.Params("id"))
    if err != nil {
        return helper.ServiceResponse(c, nil, err)
    }

    // Panggil helper untuk cek akses
    if err := helper.HandleStudentDetailAccess(c, student.UserID, userID, role); err != nil {
        return helper.ErrorResponse(c, fiber.StatusForbidden, err)
    }

    // Jika lolos, return datanya
    return helper.ServiceResponse(c, student, nil)
})

    // -----------------------------
    // 3. UPDATE Advisor
    // -----------------------------
    students.Put("/:id/advisor", rbacUpdate, func(c *fiber.Ctx) error {

        ctx, _, role, err := helper.ExtractUserContext(c)
        if err != nil {
            return helper.ErrorResponse(c, fiber.StatusUnauthorized, err)
        }

        body := struct {
            NewAdvisorID string `json:"new_advisor_id"`
        }{}
        if err := c.BodyParser(&body); err != nil {
            return helper.ErrorResponse(c, fiber.StatusBadRequest, err)
        }

        err = studentSvc.UpdateAdvisor(ctx, c.Params("id"), body.NewAdvisorID, role)
        return helper.ServiceResponse(c, nil, err)
    })

    // -----------------------------
    // 4. GET Achievements by Student
    // -----------------------------
    students.Get("/:id/achievements", func(c *fiber.Ctx) error {

        ctx, userID, role, err := helper.ExtractUserContext(c)
        if err != nil {
            return helper.ErrorResponse(c, fiber.StatusUnauthorized, err)
        }

        // cek role + kepemilikan
        if err := helper.ValidateAchievementAccess(c.Params("id"), userID, role); err != nil {
            return helper.ErrorResponse(c, fiber.StatusForbidden, err)
        }

        result, err := achievementSvc.ListAchievementsByStudentID(ctx, c.Params("id"), userID, role)
        return helper.ServiceResponse(c, result, err)
    })

}
