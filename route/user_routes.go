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

	// Otorisasi untuk fungsi Self-Access (Update/View Detail)
	// Middleware ini mengecek: Admin ATAU User ID == URL ID
	selfAccess := middleware.CanAccessSelf() 
	
	// A. List Users
	// Otorisasi: Admin Only
	// GET /api/v1/users
	users.Get("/", adminAccess, service.GetAllUsers) 

	// B. Get User Detail
	// Otorisasi: Admin atau Self-Access
	// GET /api/v1/users/:id
	users.Get("/:id", selfAccess, service.GetUserByID) 

	// C. Create User
	// Otorisasi: Admin Only
	// POST /api/v1/users
	users.Post("/", adminAccess, service.CreateNewUser) 

	// D. Update User General
	// Otorisasi: Admin atau Self-Access
	// PUT /api/v1/users/:id
	users.Put("/:id", selfAccess, service.UpdateUser) 

	// E. Delete User (Soft Delete)
	// Otorisasi: Admin Only (Walaupun Mahasiswa bisa delete diri sendiri, sebaiknya dikelola Admin)
	// DELETE /api/v1/users/:id
	users.Delete("/:id", adminAccess, service.DeleteUser) 

	// F. Update User Role
	// Otorisasi: Admin Only
	// PUT /api/v1/users/:id/role
	users.Put("/:id/role", adminAccess, service.UpdateUserRole) 
}