package routes

import (
	"database/sql" // Diperlukan untuk NewStudentRepository jika ia menerima DB
	repository "uas/app/repository/postgres"
	service "uas/app/service/postgres"

	"github.com/gofiber/fiber/v2"
)

// Asumsi: RegisterRoutes menerima koneksi DB (*sql.DB) dari main.go
// Jika Anda tidak menggunakan argumen 'db' di sini, Anda harus mengubah tanda tangan fungsi di main.go
func RegisterRoutes(app *fiber.App, db *sql.DB) { // 🛑 Saya asumsikan Anda mengembalikan db seperti yang disarankan sebelumnya

	api := app.Group("/api/v1")

	// ===== Repositori & Service =====
	
	// User Service (Misalnya untuk Profile/RBAC User)
	userRepo := repository.NewUserRepository()
	userService := service.NewUserService(userRepo)
    
	// Auth Service
	authRepo := repository.NewAuthRepository()
	authService := service.NewAuthService(authRepo)

    // 🎯 STUDENT REPO & SERVICE (Menggunakan koneksi DB yang diterima)
    studentRepo := repository.NewStudentRepository(db) // Asumsi constructor menerima *sql.DB
    studentService := service.NewStudentService(studentRepo)

	// ===== User Routes =====
	SetupUserRoutes(api, userService, authService) // kirim juga authService
    
    // ✅ DAFTARKAN ROUTE STUDENT DI SINI
    StudentRoutes(api, authService, studentService) 

	// ===== Auth Routes =====
	SetupAuthRoutes(api, authService)
}

// Catatan: Jika NewUserRepository dan NewAuthRepository membutuhkan DB,
// Anda harus mengubahnya menjadi repository.NewUserRepository(db) dan NewAuthRepository(db).
// Saya mempertahankan versi aslinya agar tetap konsisten dengan model yang Anda berikan, 
// tetapi menambahkan db untuk StudentRepository karena ia pasti butuh DB.