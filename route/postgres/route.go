package routes

import (
	"context" 
	"database/sql" // Import ini digunakan untuk tipe *sql.DB

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"

	mw "uas/middleware" // Mengacu pada package middleware

	// Menggunakan satu alias untuk package postgres repository
	postgresRepo "uas/app/repository/postgres" 
	achievementRepoMongo "uas/app/repository/mongo"
	
	authService "uas/app/service/postgres"
	studentService "uas/app/service/postgres"
	achievementService "uas/app/service/mongo"
	// Tambahkan import service Dosen
	lecturerService "uas/app/service/postgres" 
)

// Definisikan struct wrapper yang mengimplementasikan mw.TokenBlacklistChecker
// Struct ini memegang referensi ke AuthRepository yang memiliki implementasi IsBlacklisted.
type AuthBlacklistChecker struct {
	repo postgresRepo.AuthRepository
}

// Implementasi metode IsBlacklisted dengan signature yang benar.
func (c *AuthBlacklistChecker) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	// Panggil implementasi yang sebenarnya dari repository.
	return c.repo.IsBlacklisted(ctx, jti)
}
// -------------------------------------------------------------------------------------------

func RegisterRoutes(app *fiber.App, db *sql.DB, mongoClient *mongo.Client) {

	api := app.Group("/api/v1")
	
	// ===== Repository Initialization =====
	// 1. Inisialisasi Auth Repository (untuk Blacklist Checker & Auth Service)
	authRepoInst := postgresRepo.NewAuthRepository()
	
	// 2. Inisialisasi Student Repository
	studentRepoInst := postgresRepo.NewStudentRepository(db) 
	
	// 3. Inisialisasi User Repository
	userRepoInst := postgresRepo.NewUserRepository()

	// ✅ TAMBAH: Inisialisasi Lecturer Repository
	lecturerRepoInst := postgresRepo.NewLecturerRepository(db) 
	
	// 4. Buat Checker instance yang memegang Repo
	blacklistCheckerInst := &AuthBlacklistChecker{repo: authRepoInst}

	// Panggil JWTMiddleware dengan instance yang benar
	jwtMiddleware := mw.JWTMiddleware(blacklistCheckerInst) 

	// ===== Service Initialization =====
	
	// FIX 2: Gunakan userRepoInst yang baru diinisialisasi
	userService := authService.NewUserService(userRepoInst)

	// authService tetap menggunakan AuthRepo untuk fungsi otentikasi/blacklist.
	authServiceInst := authService.NewAuthService(authRepoInst)

	// Student Service menggunakan Student Repo yang diinisialisasi di atas
	studentServiceInst := studentService.NewStudentService(studentRepoInst)
	
	// ✅ TAMBAH: Inisialisasi Lecturer Service
	lecturerServiceInst := lecturerService.NewLecturerService(lecturerRepoInst)

	// ===== Achievement =====
	achievementCollection := mongoClient.Database("uas").Collection("achievements")

	mongoAchievementRepo := achievementRepoMongo.NewMongoAchievementRepository(achievementCollection)
	postgresAchievementRepo := postgresRepo.NewPostgreAchievementRepository(db) // Menggunakan alias postgresRepo

	achievementServiceInst := achievementService.NewAchievementService(
		mongoAchievementRepo,
		postgresAchievementRepo,
	)

	// ===== Routes =====
	SetupUserRoutes(api, userService, authServiceInst, jwtMiddleware)

	SetupAuthRoutes(api, authServiceInst, jwtMiddleware)

	StudentRoutes(api, authServiceInst, studentServiceInst, achievementServiceInst, jwtMiddleware)
	
	// ✅ TAMBAH: Pendaftaran Lecturer Routes
	SetupLecturerRoutes(api, authServiceInst, lecturerServiceInst, jwtMiddleware)

	ReportRoutes(api, authServiceInst, achievementServiceInst, jwtMiddleware)
}