package routes

import (
    "database/sql"

    "github.com/gofiber/fiber/v2"
    "go.mongodb.org/mongo-driver/mongo"

    // Repository
    pgRepo "uas/app/repository/postgres"
    mongoRepo "uas/app/repository/mongo"

    // Service
    authServicePkg "uas/app/service/postgres"
    studentServicePkg "uas/app/service/postgres"
    achievementServicePkg "uas/app/service/mongo"
)

// RegisterRoutesMongo inisialisasi service & kirim ke AchievementRoutes
func RegisterRoutesMongo(app *fiber.App, mongodb any, db *sql.DB) {
    api := app.Group("/api/v1")

    // =========================
    // Auth Service
    // =========================
    authRepo := pgRepo.NewAuthRepository()
    authService := authServicePkg.NewAuthService(authRepo)

    // =========================
    // Student Service
    // =========================
    studentRepo := pgRepo.NewStudentRepository(db)
    studentService := studentServicePkg.NewStudentService(studentRepo)

    // =========================
    // Mongo Repository (Achievement)
    // =========================
    coll, ok := mongodb.(*mongo.Collection)
    if !ok {
        panic("mongoColl harus bertipe *mongo.Collection")
    }
    achievementMongoRepo := mongoRepo.NewMongoAchievementRepository(coll)

    // =========================
    // Postgre Repository (Achievement)
    // =========================
    achievementPGRepo := pgRepo.NewPostgreAchievementRepository(db)

    // =========================
    // Achievement Service
    // =========================
    achievementService := achievementServicePkg.NewAchievementService(achievementMongoRepo, achievementPGRepo)

    // =========================
    // Register Routes
    // =========================
    AchievementRoutes(api, authService, achievementService, studentService)
}
