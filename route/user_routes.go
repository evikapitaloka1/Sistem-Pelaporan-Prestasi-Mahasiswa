package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)


func UsersRoutes(r fiber.Router) {
	users := r.Group("/users", middleware.Protected()) 
	adminAccess := middleware.CheckPermission("user:manage")
	selfAccess := middleware.CanAccessSelf() 
	users.Get("/", adminAccess, service.GetAllUsers) 
	users.Get("/:id", selfAccess, service.GetUserByID) 
	users.Post("/", adminAccess, service.CreateNewUser) 
	users.Put("/:id", selfAccess, service.UpdateUser) 
	users.Delete("/:id", adminAccess, service.DeleteUser) 
	users.Put("/:id/role", adminAccess, service.UpdateUserRole) 
}