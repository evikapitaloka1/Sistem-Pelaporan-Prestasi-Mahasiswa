package routes

import (
	repository "uas/app/repository/postgres"
	service "uas/app/service/postgres"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")

	userRepo := repository.NewUserRepository()
	userService := service.NewUserService(userRepo)

	SetupUserRoutes(api, userService)
}
