package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

// UsersRoutes mengelompokkan semua endpoint manajemen user
// Endpoint: /api/v1/users/*
func UsersRoutes(r fiber.Router) {
	users := r.Group("/users", middleware.Protected()) // Pasang Protected di group level
	
	// Otorisasi untuk fungsi Admin (Create, List All, Role Update, Delete)
	adminAccess := middleware.CheckPermission("user:manage")
	selfAccess := middleware.CanAccessSelf() 
	users.Get("/", adminAccess, service.GetAllUsers) 
	users.Get("/:id", selfAccess, service.GetUserByID) 
	users.Post("/", adminAccess, service.CreateNewUser) 
	users.Put("/:id", selfAccess, service.UpdateUser) 
	users.Delete("/:id", adminAccess, service.DeleteUser) 
	users.Put("/:id/role", adminAccess, service.UpdateUserRole) 
}