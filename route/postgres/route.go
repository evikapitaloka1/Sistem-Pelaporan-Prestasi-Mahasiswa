package routes

import (
	"database/sql"
	// 🛑 PERBAIKAN: Import package mongo-driver/mongo untuk tipe *mongo.Client
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo" 
	
	// Import package lokal Anda
	
	// Pastikan semua path import di bawah ini sudah benar
	authRepo "uas/app/repository/postgres"
	studentRepo "uas/app/repository/postgres"
	achievementRepoMongo "uas/app/repository/mongo"
	achievementRepoPostgres "uas/app/repository/postgres"

	authService "uas/app/service/postgres"
	studentService "uas/app/service/postgres"
	achievementService "uas/app/service/mongo"
)

// RegisterRoutes adalah fungsi utama untuk mendaftarkan semua endpoint di aplikasi Fiber.
// Ia bertanggung jawab untuk menginisialisasi repository dan service, lalu mendaftarkan routes.
func RegisterRoutes(app *fiber.App, db *sql.DB, mongoClient *mongo.Client) {
	api := app.Group("/api/v1")

	// ===== User & Auth =====
	userRepo := authRepo.NewUserRepository() // ubah jika butuh db
	userService := authService.NewUserService(userRepo)
	authRepoInst := authRepo.NewAuthRepository() // ubah jika butuh db
	authServiceInst := authService.NewAuthService(authRepoInst)

	// ===== Student =====
	studentRepoInst := studentRepo.NewStudentRepository(db)
	studentServiceInst := studentService.NewStudentService(studentRepoInst)

	// ===== Achievement (Mongo & Postgres) =====
	achievementCollection := mongoClient.Database("uas").Collection("achievements")

	mongoAchievementRepo := achievementRepoMongo.NewMongoAchievementRepository(achievementCollection)
	postgresAchievementRepo := achievementRepoPostgres.NewPostgreAchievementRepository(db)

	achievementServiceInst := achievementService.NewAchievementService(mongoAchievementRepo, postgresAchievementRepo)

	// ===== Routes Setup =====
	SetupUserRoutes(api, userService, authServiceInst)
	SetupAuthRoutes(api, authServiceInst)

	// Pendaftaran Student Routes (yang juga menggunakan achievementServiceInst)
	StudentRoutes(api, authServiceInst, studentServiceInst, achievementServiceInst)
    
	// ✅ PENDAFTARAN REPORT & ANALYTICS ROUTES
	ReportRoutes(api, authServiceInst, achievementServiceInst)
}