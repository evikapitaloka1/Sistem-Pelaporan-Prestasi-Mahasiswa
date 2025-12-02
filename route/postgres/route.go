package routes

import (
	"database/sql"
	// 🛑 PERBAIKAN: Import package mongo-driver/mongo untuk tipe *mongo.Client
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo" 

	// Import package lokal Anda


	authRepo "uas/app/repository/postgres"
	studentRepo "uas/app/repository/postgres"
	achievementRepoMongo "uas/app/repository/mongo"
	achievementRepoPostgres "uas/app/repository/postgres"

	authService "uas/app/service/postgres"
	studentService "uas/app/service/postgres"
	achievementService "uas/app/service/mongo"
)

// ✅ PERBAIKAN 1: Tambahkan mongoClient ke parameter fungsi
// Pastikan di main.go Anda sekarang memanggil: RegisterRoutes(app, db, mongoClient)
func RegisterRoutes(app *fiber.App, db *sql.DB, mongoClient *mongo.Client) {
	api := app.Group("/api/v1")

	// ===== User & Auth =====
	userRepo := authRepo.NewUserRepository() 		// ubah jika butuh db
	userService := authService.NewUserService(userRepo)
	authRepoInst := authRepo.NewAuthRepository() 	// ubah jika butuh db
	authServiceInst := authService.NewAuthService(authRepoInst)

	// ===== Student =====
	studentRepoInst := studentRepo.NewStudentRepository(db)
	studentServiceInst := studentService.NewStudentService(studentRepoInst)

	// ===== MongoDB Achievement (Dihilangkan dari sini, diasumsikan sudah terkoneksi di main.go) =====
    // ✅ PERBAIKAN 2: Gunakan mongoClient yang diterima
	achievementCollection := mongoClient.Database("uas").Collection("achievements")

	mongoAchievementRepo := achievementRepoMongo.NewMongoAchievementRepository(achievementCollection)
	postgresAchievementRepo := achievementRepoPostgres.NewPostgreAchievementRepository(db)

	achievementServiceInst := achievementService.NewAchievementService(mongoAchievementRepo, postgresAchievementRepo)

	// ===== Routes Setup =====
    // Catatan: Pastikan fungsi SetupUserRoutes dan SetupAuthRoutes sudah ada
	SetupUserRoutes(api, userService, authServiceInst)
	SetupAuthRoutes(api, authServiceInst)

	// ✅ Student Routes, sudah lengkap semua service
	StudentRoutes(api, authServiceInst, studentServiceInst, achievementServiceInst)
}