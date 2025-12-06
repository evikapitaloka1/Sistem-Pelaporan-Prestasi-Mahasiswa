package routes

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"

    postgres "uas/app/service/postgres"
    model "uas/app/model/postgres"
    mw "uas/middleware"
    "uas/helper"
)

func SetupUserRoutes(
    router fiber.Router,
    userService postgres.IUserService,
    authService postgres.IAuthService,
    jwtMiddleware fiber.Handler,
) {
    // RBAC
    rbacManage := mw.RBACMiddleware("user:manage", authService)

    // Group route
    users := router.Group("/users", jwtMiddleware)

    // ============================================================
    // 1. GET ALL USERS
    // ============================================================
    users.Get("/", rbacManage, func(c *fiber.Ctx) error {

        result, err := userService.GetAllUsers(c.Context())
        return helper.ServiceResponse(c, result, err)
    })

    // ============================================================
    // 2. GET USER BY ID
    // ============================================================
    users.Get("/:id", rbacManage, func(c *fiber.Ctx) error {

        userID, err := uuid.Parse(c.Params("id"))
        if err != nil {
            return helper.SendError(c, fiber.StatusBadRequest, "Invalid user ID")
        }

        result, err := userService.GetUserByID(c.Context(), userID)
        return helper.ServiceResponse(c, result, err)
    })

    // ============================================================
    // 3. CREATE USER
    // ============================================================
    users.Post("/", rbacManage, func(c *fiber.Ctx) error {

        var req model.CreateUserRequest
        if err := c.BodyParser(&req); err != nil {
            return helper.SendError(c, fiber.StatusBadRequest, err.Error())
        }

        newID, err := userService.CreateUser(c.Context(), req)
        if err != nil {
            return helper.ServiceResponse(c, nil, err)
        }

        return helper.Created(c, fiber.Map{"id": newID})
    })

    // ============================================================
    // 4. UPDATE USER
    // ============================================================
    users.Put("/:id", rbacManage, func(c *fiber.Ctx) error {

        userID, err := uuid.Parse(c.Params("id"))
        if err != nil {
            return helper.SendError(c, fiber.StatusBadRequest, "Invalid user ID")
        }

        var req model.UpdateUserRequest
        if err := c.BodyParser(&req); err != nil {
            return helper.SendError(c, fiber.StatusBadRequest, err.Error())
        }

        err = userService.UpdateUser(c.Context(), userID, req)
        return helper.ServiceResponse(c, nil, err)
    })

    // ============================================================
    // 5. DELETE USER
    // ============================================================
    users.Delete("/:id", rbacManage, func(c *fiber.Ctx) error {

        userID, err := uuid.Parse(c.Params("id"))
        if err != nil {
            return helper.SendError(c, fiber.StatusBadRequest, "Invalid user ID")
        }

        err = userService.DeleteUser(c.Context(), userID)
        return helper.ServiceResponse(c, nil, err)
    })
	    // ============================================================
    // 6. UPDATE USER ROLE
    // ============================================================
    users.Put("/:id/role", rbacManage, func(c *fiber.Ctx) error {

        userID, err := uuid.Parse(c.Params("id"))
        if err != nil {
            return helper.SendError(c, fiber.StatusBadRequest, "Invalid user ID")
        }

        body := struct {
            RoleID string `json:"role_id"`
        }{}

        if err := c.BodyParser(&body); err != nil {
            return helper.SendError(c, fiber.StatusBadRequest, err.Error())
        }

        if body.RoleID == "" {
            return helper.SendError(c, fiber.StatusBadRequest, "role_id is required")
        }

        roleUUID, err := uuid.Parse(body.RoleID)
        if err != nil {
            return helper.SendError(c, fiber.StatusBadRequest, "Invalid role_id")
        }

        err = userService.UpdateUserRole(c.Context(), userID, roleUUID)
        return helper.ServiceResponse(c, nil, err)
    })

}
