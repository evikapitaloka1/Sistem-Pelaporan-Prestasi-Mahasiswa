package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware" // Middleware Protected ada di sini

	"github.com/gofiber/fiber/v2"
)

// UsersRoutes mengelompokkan semua endpoint manajemen user
// Endpoint: /api/v1/users/*
func UsersRoutes(r fiber.Router) {
	// Grouping URL: /api/v1/users
	users := r.Group("/users") 

	// Middleware akses: Hanya memerlukan login (Protected), tanpa permission check spesifik.
	// Jika non-admin mencoba mengakses, sistem akan mengizinkan.
	
	protected := middleware.Protected()
	
	// A. List Users
	// GET /api/v1/users
	users.Get("/", protected, service.GetAllUsers) 

	// B. Get User Detail
	// GET /api/v1/users/:id
	users.Get("/:id", protected, service.GetUserByID) 

	// C. Create User
	// POST /api/v1/users
	users.Post("/", protected, service.CreateNewUser) 

	// D. Update User General
	// PUT /api/v1/users/:id
	users.Put("/:id", protected, service.UpdateUser) 

	// E. Delete User (Soft Delete)
	// DELETE /api/v1/users/:id
	users.Delete("/:id", protected, service.DeleteUser) 

	// F. Update User Role
	// PUT /api/v1/users/:id/role
	users.Put("/:id/role", protected, service.UpdateUserRole) 
}